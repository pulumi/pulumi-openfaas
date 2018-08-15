import * as pulumi from "@pulumi/pulumi";

/**
 * Provides an OpenFaaS Function resource.
 */
export class Function extends pulumi.CustomResource {
    /**
     * Get an existing Function resource's state with the given name, ID, and optional extra
     * properties used to qualify the lookup.
     *
     * @param name The _unique_ name of the resulting resource.
     * @param id The _unique_ provider ID of the resource to lookup.
     * @param state Any extra arguments used during the lookup.
     */
    public static get(name: string, id: pulumi.Input<pulumi.ID>, state?: FunctionState): Function {
        return new Function(name, <any>state, { id });
    }

    public readonly service: pulumi.Output<string>;
    public readonly network: pulumi.Output<string> | undefined;
    public readonly image: pulumi.Output<string>;
    public readonly envProcess: pulumi.Output<string>;
    public readonly envVars: pulumi.Output<{[key: string]: string}> | undefined;
    public readonly labels: pulumi.Output<string[]> | undefined;
    public readonly annotations: pulumi.Output<string[]> | undefined;
    public readonly registryAuth: pulumi.Output<string> | undefined;

    /**
     * Create a Function resource with the given unique name, arguments, and options.
     *
     * @param name The _unique_ name of the resource.
     * @param args The arguments to use to populate this resource's properties.
     * @param opts A bag of options that control this resource's behavior.
     */
    constructor(name: string, args: FunctionArgs, opts?: pulumi.ResourceOptions)
    constructor(name: string, argsOrState?: FunctionArgs | FunctionState, opts?: pulumi.ResourceOptions) {
        let inputs: pulumi.Inputs = {};
        if (opts && opts.id) {
            const state = argsOrState as FunctionState | undefined;
            inputs["service"] = state ? state.service : undefined;
            inputs["network"] = state ? state.network : undefined;
            inputs["image"] = state ? state.image : undefined;
            inputs["envProcess"] = state ? state.envProcess : undefined;
            inputs["envVars"] = state ? state.envVars : undefined;
            inputs["labels"] = state ? state.labels : undefined;
            inputs["annotations"] = state ? state.annotations : undefined;
            inputs["registryAuth"] = state ? state.registryAuth : undefined;
        } else {
            const args = argsOrState as FunctionArgs | undefined;
            if (!args || args.service === undefined) {
                throw new Error("Missing required property 'service'");
            }
            if (!args || args.image === undefined) {
                throw new Error("Missing required property 'image'");
            }
            inputs["service"] = args ? args.service : undefined;
            inputs["network"] = args ? args.network : undefined;
            inputs["image"] = args ? args.image : undefined;
            inputs["envProcess"] = args ? args.envProcess : undefined;
            inputs["envVars"] = args ? args.envVars : undefined;
            inputs["labels"] = args ? args.labels : undefined;
            inputs["annotations"] = args ? args.annotations : undefined;
            inputs["registryAuth"] = args ? args.registryAuth : undefined;
        }
        super("openfaas:system:Function", name, inputs, opts);
    }
}

/**
 * Input properties used for looking up and filtering Function resources.
 */
export interface FunctionState {
    readonly service?: pulumi.Input<string>;
    readonly network?: pulumi.Input<string>;
    readonly image?: pulumi.Input<string>;
    readonly envProcess?: pulumi.Input<string>;
    readonly envVars?: pulumi.Input<{[key: string]: pulumi.Input<string>}>;
    readonly labels?: pulumi.Input<pulumi.Input<string>[]>;
    readonly annotations?: pulumi.Input<pulumi.Input<string>[]>;
    readonly registryAuth?: pulumi.Input<string>;
}

/**
 * The set of arguments for constructing a Function resource.
 */
export interface FunctionArgs {
    readonly service: pulumi.Input<string>;
    readonly network?: pulumi.Input<string>;
    readonly image: pulumi.Input<string>;
    readonly envProcess?: pulumi.Input<string>;
    readonly envVars?: pulumi.Input<{[key: string]: pulumi.Input<string>}>;
    readonly labels?: pulumi.Input<pulumi.Input<string>[]>;
    readonly annotations?: pulumi.Input<pulumi.Input<string>[]>;
    readonly registryAuth?: pulumi.Input<string>;
}
