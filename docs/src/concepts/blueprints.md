# Blueprints

A blueprint contains all of the configuration data necessary for Catalyst Forge to process a [project](./projects.md).
The configuration is defined in a blueprint file (`blueprint.cue`) that exists in the root directory of a project.
Blueprints are _not_ optional and must exist for every project.

## Language

Blueprints are written in the [CUE](https://cuelang.org/) language (denoted by the `.cue` extension).
When a project is loaded, the Forge runtime will automatically load and parse the associated blueprint file using CUE.
For more information on CUE, please refer to the [CUE documentation](https://cuelang.org/docs/).

## Schema

The schema for blueprint files is defined in both Go and CUE.
The latest schema can be found
[here](https://github.com/input-output-hk/catalyst-forge/blob/master/lib/blueprint/schema/_embed/schema.cue){:target="_blank"}.
Alternatively, the Go code responsible for generating the schema can be explored
[here](https://godocs.io/github.com/input-output-hk/catalyst-forge/lib/blueprint/schema){:target="_blank"}.

Note that the schema is enforced at runtime.
Improperly named fields, fields not specified in the schema, or incorrect types on fields will all cause runtime errors in all
Forge systems.

### Versioning

!!! note

    Blueprints are versioned using [semantic versioning](https://semver.org/).
    However, only the major and minor sections are used.
    The patch section is ignored.

Every blueprint must specify a `version` field at the top-level of the file.
This informs Forge what version of the schema is being used.
The version is [hard-coded](https://github.com/input-output-hk/catalyst-forge/blob/master/lib/blueprint/schema/version.cue) in
the source code and is bumped whenever a change to the schema is made.

All systems within the Forge ecosystem have a dependency on this schema.
For exampe, the version embedded into the Forge CLI can be found by running:

```shell
$ forge version
forge version v0.1.0 linux/amd64
config schema version 1.0.0
```

When a Forge system is processing a blueprint file, it first checks the version specified in the file against the version it was
compiled with.
It then uses the following rules to determine if it is safe to proceed:

- The major versions match
- The minor version of the embedded schema is _greater than or equal to_ the minor verion specified in the blueprint

In the case where the major versions mismatch, the tool will refuse to parse the blueprint file.
In the case where the minor version is less than the one specified in the blueprint, a warning is emitted but the blueprint file
will still be parsed.
