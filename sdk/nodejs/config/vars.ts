// *** WARNING: this file was generated by the Pulumi Terraform Bridge (tfgen) Tool. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";

let __config = new pulumi.Config("openfaas");

/**
 * The URL of the OpenFaaS API gateway.
 */
export let endpoint: string = __config.require("endpoint");
