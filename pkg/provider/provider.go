// Copyright 2016-2018, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	pbempty "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/pulumi/pulumi/pkg/resource/plugin"
	"github.com/pulumi/pulumi/pkg/util/rpcutil/rpcerror"
	pulumirpc "github.com/pulumi/pulumi/sdk/proto/go"
	"google.golang.org/grpc/codes"

	"github.com/pulumi/pulumi-openfaas/pkg/client"
)

type cancellationContext struct {
	context context.Context
	cancel  context.CancelFunc
}

func makeCancellationContext() *cancellationContext {
	var ctx, cancel = context.WithCancel(context.Background())
	return &cancellationContext{
		context: ctx,
		cancel:  cancel,
	}
}

type faasProvider struct {
	canceler *cancellationContext
	client   *client.Client
	name     string
	version  string
}

func makeFaasProvider(name, version string) (pulumirpc.ResourceProviderServer, error) {
	return &faasProvider{
		canceler: makeCancellationContext(),
		name:     name,
		version:  version,
	}, nil
}

func (p *faasProvider) label() string {
	return fmt.Sprintf("Provider[%s]", p.name)
}

// Configure configures the resource provider with "globals" that control its behavior.
func (p *faasProvider) Configure(_ context.Context, req *pulumirpc.ConfigureRequest) (*pbempty.Empty, error) {
	const faasNamespace = "openfaas:config:"

	vars := req.GetVariables()

	endpoint, ok := vars[faasNamespace+"endpoint"]
	if !ok {
		missingKey := &pulumirpc.ConfigureErrorMissingKeys_MissingKey{
			Name:        "openfaas:config:endpoint",
			Description: "the endpoint of the OpenFaaS API gateway",
		}

		err := rpcerror.New(codes.InvalidArgument, "required configuration keys were missing")

		// Clients of our RPC endpoint will be looking for this detail in order to figure out
		// which keys need descriptive error messages.
		return nil, rpcerror.WithDetails(err, &pulumirpc.ConfigureErrorMissingKeys{
			MissingKeys: []*pulumirpc.ConfigureErrorMissingKeys_MissingKey{missingKey},
		})
	}

	username, password := vars[faasNamespace+"username"], vars[faasNamespace+"password"]

	p.client = client.NewClient(http.DefaultClient, endpoint, username, password)

	return &pbempty.Empty{}, nil
}

// Invoke dynamically executes a built-in function in the provider.
func (p *faasProvider) Invoke(context.Context, *pulumirpc.InvokeRequest) (*pulumirpc.InvokeResponse, error) {
	panic("Invoke not implemented")
}

type function struct {
	Service      string            `pulumi:"service,forceNew"`
	Network      string            `pulumi:"network,optional"`
	Image        string            `pulumi:"image"`
	EnvProcess   string            `pulumi:"envProcess,optional"`
	EnvVars      map[string]string `pulumi:"envVars,optional"`
	Labels       []string          `pulumi:"labels,optional"`
	Annotations  []string          `pulumi:"annotations,optional"`
	Secrets      []string          `pulumi:"secrets,optional"`
	RegistryAuth string            `pulumi:"registryAuth,optional"`
}

const functionType = "openfaas:system:Function"

// Check validates that the given property bag is valid for a resource of the given type and returns
// the inputs that should be passed to successive calls to Diff, Create, or Update for this
// resource. As a rule, the provider inputs returned by a call to Check should preserve the original
// representation of the properties as present in the program inputs. Though this rule is not
// required for correctness, violations thereof can negatively impact the end-user experience, as
// the provider inputs are using for detecting and rendering diffs.
func (p *faasProvider) Check(ctx context.Context, req *pulumirpc.CheckRequest) (*pulumirpc.CheckResponse, error) {
	urn := resource.URN(req.GetUrn())
	label := fmt.Sprintf("%s.Check(%s)", p.label(), urn)
	glog.V(9).Infof("%s executing", label)

	if urn.Type() != functionType {
		return nil, errors.Errorf("unknown resource type %v", urn.Type())
	}

	news, err := plugin.UnmarshalProperties(req.GetNews(), plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.news", label), KeepUnknowns: true, SkipNulls: true,
	})
	if err != nil {
		return nil, err
	}

	// Check the schema.
	failures, err := checkProperties(news, function{})
	if err != nil {
		return nil, err
	}

	// We currently don't change the inputs during check.
	return &pulumirpc.CheckResponse{Inputs: req.GetNews(), Failures: failures}, nil
}

// Diff checks what impacts a hypothetical update will have on the resource's properties.
func (p *faasProvider) Diff(ctx context.Context, req *pulumirpc.DiffRequest) (*pulumirpc.DiffResponse, error) {
	urn := resource.URN(req.GetUrn())
	label := fmt.Sprintf("%s.Diff(%s)", p.label(), urn)
	glog.V(9).Infof("%s executing", label)

	if urn.Type() != functionType {
		return nil, errors.Errorf("unknown resource type %v", urn.Type())
	}

	olds, err := plugin.UnmarshalProperties(req.GetOlds(), plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.news", label), KeepUnknowns: true, SkipNulls: true,
	})
	if err != nil {
		return nil, err
	}

	news, err := plugin.UnmarshalProperties(req.GetNews(), plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.news", label), KeepUnknowns: true, SkipNulls: true,
	})
	if err != nil {
		return nil, err
	}

	// Diff the values.
	changed, replaces, err := diffProperties(olds, news, function{})
	if err != nil {
		return nil, err
	}

	diff := pulumirpc.DiffResponse_DIFF_NONE
	if changed {
		diff = pulumirpc.DiffResponse_DIFF_SOME
	}

	return &pulumirpc.DiffResponse{
		Changes:             diff,
		Replaces:            replaces,
		Stables:             []string{},
		DeleteBeforeReplace: false,
	}, nil
}

// Create allocates a new instance of the provided resource and returns its unique ID afterwards.
// (The input ID must be blank.)  If this call fails, the resource must not have been created (i.e.,
// it is "transacational").
func (p *faasProvider) Create(ctx context.Context, req *pulumirpc.CreateRequest) (*pulumirpc.CreateResponse, error) {
	urn := resource.URN(req.GetUrn())
	label := fmt.Sprintf("%s.Create(%s)", p.label(), urn)
	glog.V(9).Infof("%s executing", label)

	if urn.Type() != functionType {
		return nil, errors.Errorf("unknown resource type %v", urn.Type())
	}

	newResInputs, err := plugin.UnmarshalProperties(req.GetProperties(), plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.properties", label), KeepUnknowns: true, SkipNulls: true,
	})
	if err != nil {
		return nil, err
	}

	var f function
	if err := decodeProperties(newResInputs, &f); err != nil {
		return nil, err
	}

	clientFunc := &client.Function{
		Service:      f.Service,
		Network:      f.Network,
		Image:        f.Image,
		EnvProcess:   f.EnvProcess,
		EnvVars:      f.EnvVars,
		Labels:       f.Labels,
		Annotations:  f.Annotations,
		Secrets:      f.Secrets,
		RegistryAuth: f.RegistryAuth,
	}

	if err := p.client.CreateFunction(p.canceler.context, clientFunc); err != nil {
		return nil, err
	}

	return &pulumirpc.CreateResponse{
		Id: f.Service, Properties: req.GetProperties(),
	}, nil
}

// Read the current live state associated with a resource.  Enough state must be include in the
// inputs to uniquely identify the resource; this is typically just the resource ID, but may also
// include some properties.
func (p *faasProvider) Read(ctx context.Context, req *pulumirpc.ReadRequest) (*pulumirpc.ReadResponse, error) {
	urn := resource.URN(req.GetUrn())
	label := fmt.Sprintf("%s.Update(%s)", p.label(), urn)
	glog.V(9).Infof("%s executing", label)

	if urn.Type() != functionType {
		return nil, errors.Errorf("unknown resource type %v", urn.Type())
	}

	f, err := p.client.GetFunction(p.canceler.context, req.GetId())
	if err != nil {
		return nil, err
	}

	// TODO: encode response
	props, err := encodeProperties(function{
		Service:      f.Service,
		Network:      f.Network,
		Image:        f.Image,
		EnvProcess:   f.EnvProcess,
		EnvVars:      f.EnvVars,
		Labels:       f.Labels,
		Annotations:  f.Annotations,
		Secrets:      f.Secrets,
		RegistryAuth: f.RegistryAuth,
	})
	if err != nil {
		return nil, err
	}

	outputs, err := plugin.MarshalProperties(props, plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.outputs", label), KeepUnknowns: true, SkipNulls: true,
	})
	if err != nil {
		return nil, err
	}

	return &pulumirpc.ReadResponse{Id: f.Service, Properties: outputs}, nil
}

// Update updates an existing resource with new values.
func (p *faasProvider) Update(ctx context.Context, req *pulumirpc.UpdateRequest) (*pulumirpc.UpdateResponse, error) {
	urn := resource.URN(req.GetUrn())
	label := fmt.Sprintf("%s.Update(%s)", p.label(), urn)
	glog.V(9).Infof("%s executing", label)

	if urn.Type() != functionType {
		return nil, errors.Errorf("unknown resource type %v", urn.Type())
	}

	newResInputs, err := plugin.UnmarshalProperties(req.GetNews(), plugin.MarshalOptions{
		Label: fmt.Sprintf("%s.properties", label), KeepUnknowns: true, SkipNulls: true,
	})
	if err != nil {
		return nil, err
	}

	var f function
	if err := decodeProperties(newResInputs, &f); err != nil {
		return nil, err
	}

	clientFunc := &client.Function{
		Service:      f.Service,
		Network:      f.Network,
		Image:        f.Image,
		EnvProcess:   f.EnvProcess,
		EnvVars:      f.EnvVars,
		Labels:       f.Labels,
		Annotations:  f.Annotations,
		Secrets:      f.Secrets,
		RegistryAuth: f.RegistryAuth,
	}

	if err := p.client.UpdateFunction(p.canceler.context, clientFunc); err != nil {
		return nil, err
	}

	return &pulumirpc.UpdateResponse{Properties: req.GetNews()}, nil
}

// Delete tears down an existing resource with the given ID.  If it fails, the resource is assumed
// to still exist.
func (p *faasProvider) Delete(ctx context.Context, req *pulumirpc.DeleteRequest) (*pbempty.Empty, error) {
	urn := resource.URN(req.GetUrn())
	label := fmt.Sprintf("%s.Delete(%s)", p.label(), urn)
	glog.V(9).Infof("%s executing", label)

	if urn.Type() != functionType {
		return nil, errors.Errorf("unknown resource type %v", urn.Type())
	}

	if err := p.client.DeleteFunction(p.canceler.context, req.GetId()); err != nil {
		return nil, err
	}

	return &pbempty.Empty{}, nil
}

// GetPluginInfo returns generic information about this plugin, like its version.
func (p *faasProvider) GetPluginInfo(context.Context, *pbempty.Empty) (*pulumirpc.PluginInfo, error) {
	return &pulumirpc.PluginInfo{
		Version: p.version,
	}, nil
}

// Cancel signals the provider to gracefully shut down and abort any ongoing resource operations.
func (p *faasProvider) Cancel(context.Context, *pbempty.Empty) (*pbempty.Empty, error) {
	p.canceler.cancel()
	return &pbempty.Empty{}, nil
}
