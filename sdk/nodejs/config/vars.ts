import * as pulumi from "@pulumi/pulumi";

let __config = new pulumi.Config("openfaas");

/**
 * The URL of the OpenFaaS API gateway.
 */
export let endpoint = __config.get("endpoint");

/**
 * The username (if any) to use when authenticating with the OpennFaaS API gateway.
 */
export let username = __config.get("username");

/**
 * The password (if any) to use when authenticating with the OpennFaaS API gateway.
 */
export let password = __config.get("password");

/**
 * Whether or not to disable TLS verification when connecting to the OpenFaaS API gateway. Defaults to false.
 */
export let tlsSkipVerify = __config.get("tlsSkipVerify");
