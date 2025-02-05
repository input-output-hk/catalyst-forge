# Base Deployment Module

This directory contains the base [KCL](https://www.kcl-lang.io/) module used for developing deployment modules.
When Catalyst Forge executes a module, it passes data in a very particular format as inputs to the module.
The `base` KCL module provides wrappers and types that help to bridge what Forge passes to what a module developer needs.
See the [examples](../examples/) folder for examples on using this module.

## Installing

You can add the module to your existing KCL module with the following command:

```
kcl mod add --oci ghcr.io/input-output-hk/catalyst-forge/base
```

Updates can be performed by running:

```
kcl mod update
```

## Reading

```kcl
import base.schemas
import base.util

schema App(schemas.App):
    image: str
    replicas: int = 1
    port: int

app: App { **util.Read() }
```

The `base` module imposes a light structure on how module inputs should be consumed.
The module assumes that you've created a type that inherits from `schemas.App` which is used to hold input.

Any additional fields added to the new type are inferred to exist in the module's `values` input.
For example, we can consume the above definition with the following forge module:

```cue
{
	main: {
		instance:  "test"
		namespace: "default"
		path:      "."
		values: {
			image:    "nginx"
			replicas: 2
			port:     80
		}
	}
}
```

And then execute it with:

```
forge mod template mod.cue
```

The `image`, `replicas`, and `port` fields are automatically populated into the `App` type.
Required fields that are missing in the `values`, or non-existent fields in the `values`, will result in an error.

## Streaming

```kcl
import base.schemas

schema AppInstance(schemas.Instance):
    app: specs.App
```

The `base` module assumes you'll have a "instance" type that inherits from `schemas.Instance`.
The child type will inherit a field called `manifests` where all generated Kubernetes objects should be placed.
To stream all YAML documents from the `manifests` field, use the `stream()` function:

```
import base.util

util.Stream(
    AppInstance {
        app: specs.App { **util.Read() }
    }
)
```

Forge expects the module to return a YAML stream of Kubernetes objects to apply to the cluster.
It's recommended you use the above steps to produce this, however, it's ultimately optional.