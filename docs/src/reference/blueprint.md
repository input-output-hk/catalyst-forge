# Blueprints

A blueprint file is the native configuration file format for Catalyst Forge.
They may appear in various places in a repository and are always denoted by their filename: `blueprint.cue`.

## Language

Blueprints are written in the [CUE](https://cuelang.org/) language (denoted by the `.cue` extension).
They are parsed by the CUE Go API at runtime and must meet all language restrictions.
Blueprint files can be parsed using the `cue` CLI, however, Forge provides a limited number of custom attributes that will not
be recognized by the CLI (see the [section below](#custom-attributes)).

Blueprint files must be _concrete_ at runtime.
Meaning, all fields must have _known_ values when Forge parses the blueprint file.
In cases where attributes are used to insert dynamic data, the data must be available at runtime, otherwise Forge will fail to
compile the blueprint.

### Custom Attributes

Forge currently provides two custom [attributes](https://cuelang.org/docs/reference/spec/#attributes) that can be used in blueprint
files.

#### Env

The `env` attribute can be used to pull values from an environment variable at runtime.
When Forge parses a blueprint file, it scans for this attribute and uses the fields to dynamically insert data from the environment.
If the specified environment variable is unset, a value will _not_ be set.

Example usage:

```cue
project: {
    name: "foo"
    ci: targets: {
		docker: {
			args: {
				foo: string | *"bar" @env(name="FOO",type="string")
			}
		}
    }
}
```

In the above example, the `foo` field will be set with to value present in the `FOO` environment variable.
The value in the `FOO` environment variable must be a valid string.
If the `FOO` environment variable is unset, the `foo` field will default to `bar`.

#### Forge

The `forge` attribute can be used to access runtime data collected and provided by Forge.
Not all runtime data is guaranteed to be present as some is contextual.
The below table documents all runtime data available in the latest version of Forge:

| Name              | Description                               | Type   | Context                             |
| ----------------- | ----------------------------------------- | ------ | ----------------------------------- |
| `GIT_COMMIT_HASH` | The commit hash for the current commit    | string | Always available                    |
| `GIT_TAG`         | The full name of the current git tag      | string | Available when a git tag is present |
| `GIT_TAG_VERSION` | The version suffix of the current git tag | string | Available when a git tag is present |

Example usage:

!!! note
    When using the `forge` attribute, if the target field has a different type than the one specified in the above table, the
    blueprint will fail to validate.

```cue
project: {
    name: "foo"
    ci: targets: {
		docker: {
			args: {
				version: string | *"dev" @forge(name="GIT_TAG_VERSION")
			}
		}
    }
}
```

In the above example, the `version` field will set to the version suffix of the current git tag (e.g., `v1.0.0` for
`project/v1.0.0`).
If no tag is detected, the value will default to `dev`.

### Schema

The schema for blueprint files is defined in both Go and CUE.
The latest schema can be found
[here](https://github.com/input-output-hk/catalyst-forge/blob/master/lib/schema/blueprint){:target="_blank"}.
Alternatively, the Go code responsible for generating the schema can be explored
[here](https://godocs.io/github.com/input-output-hk/catalyst-forge/lib/schema/blueprint){:target="_blank"}.

Note that the schema is enforced at runtime.
Improperly named fields, fields not specified in the schema, or incorrect types on fields will all cause runtime errors in all
Forge systems.

## Types

There are two types of blueprint files: _project_ and _global_.

### Project

A project blueprint file is responsible for defining the configuration of a project.
By convention, it is located at the root of the project directory in a repository, usually next to an `Earthfile`.
All projects _must_ be accompanied by a project blueprint file.

A project blueprint is usually denoted by the existence of the `project` field:

```cue
project: {
    name: "project"
}
```

There are several different options for configuring a project.
Please refer to the [schema](#schema) for an exhaustive list.

### Global

A global blueprint file is located at the root of the git repository and defines repository-wide configuration options.
Every time Forge runs on a project, it also searches for and unifies the global blueprint with the local project blueprint.
This ensures that every execution uses the configuration options defined in both.

A global blueprint is usually denoted by the existence of the `global` field:

```cue
global: repo: {
    name: "my-org/my-repo"
}
```

There are several different global options available.
Please refer to the [schema](#schema) for an exhaustive list.

## Loading and Discovery

When a project is loaded, the following occurs:

1. A `blueprint.cue` is searched for at the given project path and parsed/loaded.
2. A recursive upward search is performed to find the root of the git repository.
3. Once the git root is found, a `blueprint.cue` is searched for and parsed/loaded.
4. Both blueprint files are unified and used as the final configuration value.

It is an error for a `blueprint.cue` to exist outside of a git repository.
If a git root cannot be found, then the blueprint loading process will fail.
A global blueprint is optional and not required, although many features of Forge rely on the configuration options it provides.
