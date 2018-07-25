PROJECT_NAME := Pulumi OpenFaaS Resource Provider
include build/common.mk

PACK             := openfaas
PACKDIR          := sdk
PROJECT          := github.com/pulumi/pulumi-openfaas
NODE_MODULE_NAME := @pulumi/openfaas

PROVIDER        := pulumi-resource-${PACK}
CODEGEN         := pulumi-gen-${PACK}
VERSION         := $(shell scripts/get-version)

VERSION_FLAGS   := -ldflags "-X github.com/pulumi/pulumi-openfaas/pkg/version.Version=${VERSION}"

GO              ?= go
GOMETALINTERBIN ?= gometalinter
GOMETALINTER    :=${GOMETALINTERBIN} --config=Gometalinter.json
CURL            ?= curl

TESTPARALLELISM := 10
TESTABLE_PKGS   := ./pkg/... ./examples

build::
	$(GO) install $(VERSION_FLAGS) $(PROJECT)/cmd/$(PROVIDER)
	cd ${PACKDIR}/nodejs/ && \
		yarn install && \
		yarn run tsc
	cp README.md LICENSE ${PACKDIR}/nodejs/package.json ${PACKDIR}/nodejs/yarn.lock ${PACKDIR}/nodejs/bin/

lint::
	$(GOMETALINTER) ./cmd/... ./pkg/... | sort ; exit "$${PIPESTATUS[0]}"

install::
	GOBIN=$(PULUMI_BIN) $(GO) install $(VERSION_FLAGS) $(PROJECT)/cmd/$(PROVIDER)
	[ ! -e "$(PULUMI_NODE_MODULES)/$(NODE_MODULE_NAME)" ] || rm -rf "$(PULUMI_NODE_MODULES)/$(NODE_MODULE_NAME)"
	mkdir -p "$(PULUMI_NODE_MODULES)/$(NODE_MODULE_NAME)"
	cp -r sdk/nodejs/bin/. "$(PULUMI_NODE_MODULES)/$(NODE_MODULE_NAME)"
	rm -rf "$(PULUMI_NODE_MODULES)/$(NODE_MODULE_NAME)/node_modules"
	cd "$(PULUMI_NODE_MODULES)/$(NODE_MODULE_NAME)" && \
		yarn install --offline --production && \
		(yarn unlink > /dev/null 2>&1 || true) && \
		yarn link

test_all::
	PATH=$(PULUMI_BIN):$(PATH) $(GO) test -v -cover -timeout 1h -parallel ${TESTPARALLELISM} $(TESTABLE_PKGS)

.PHONY: publish_tgz
publish_tgz:
	$(call STEP_MESSAGE)
	./scripts/publish_tgz.sh

.PHONY: publish_packages
publish_packages:
	$(call STEP_MESSAGE)
	./scripts/publish_packages.sh

# The travis_* targets are entrypoints for CI.
.PHONY: travis_cron travis_push travis_pull_request travis_api
travis_cron: all
travis_push: only_build publish_tgz only_test publish_packages
travis_pull_request: all
travis_api: all
