# Deployments

Every project can be configured to be automatically deployed to Catalyst's development cluster.
This allows previewing the behavior of a project after merging changes.

## Background

Deployments are opinionated in that they are configured to work with a certain tech stack.
Specifically, application deployments are written using [Timoni](https://timoni.sh/), which uses CUE as the configuration language.
Projects will generate a [bundle file](https://timoni.sh/concepts/#bundle) from the blueprint that deploys one or more
[modules](https://timoni.sh/concepts/#module).
This bundle file is written to the configured GitOps repository where it's expected that
[Argo CD](https://argo-cd.readthedocs.io/en/stable/) will be responsible for reconciling it to a Kubernetes cluster.

## How it Works

When the CI pipeline runs, Forge will automatically scan and identify all projects that contain a `deployment` configuration block.
At the end of the pipeline, after all releases have been executed, all projects with deployments will be deployed.
Forge will use the `deployment` block to generate a Timoni bundle file.
The GitOps repository (configured in `global.deployment`) is then cloned locally and the bundle file is placed in specific location
within the repository.
The changes are then committed and pushed back to the repository.
This will trigger the GitOps operator (Argo CD) to consume the bundle file, generating and applying Kubernetes resources to the
configured cluster.

## Configuration

The deployment behavior can be specified using the `deployment` block in the project configuration.
For example:

```cue
project: {
    deployment: {
		environment: "dev"
		modules: main: {
			container: "foundry-api-deployment"
			version:   "0.1.0"
			values: {
				environment: name: "dev"
				server: image: {
					tag: _ @forge(name="GIT_COMMIT_HASH")
				}
			}
		}
	}
}
```

The following section will break down this example configuration.

### Environment

The `environment` field specifies which environment to deploy the project to.
In the current version of Forge, `dev` is the only allowed value in this field (and is also the default value).

### Modules

Deployments can consist of one or more modules, of which one module is designated as the _main_ module.
All modules share the same configuration fields:

| Name        | Description                                         | Type   | Required | Default                     |
| ----------- | --------------------------------------------------- | ------ | -------- | --------------------------- |
| `container` | The name of the container holding the Timoni module | string | no       | `[project_name]-deployment` |
| `namespace` | The kubernetes namespace to deploy to               | string | no       | `default`                   |
| `values`    | The configuration values to pass to the module      | Object | no       | `{}`                        |
| `version`   | The version of the container to use                 | string | yes      | N/A                         |

#### Main Module

The _main_ module is the Timoni module that is responsible for deploying the primary service managed by the project.
For example, an API server's main module would configure all of the necessary Kubernetes resources required for running the server
(like a deployment and service).

#### Support Modules

Support modules are modules that provide supplementary resources that the project may require.
For example, a project may need access to a database.
A support module can be configured to point to a Timoni module that will ensure a database is setup for the project to use.

In the current version of Forge, this field is not used, as no support modules exist at the time of this writing.

### Values

Most Timoni modules require values to be passed in order to configure how the module goes about deploying the project.
For example, most modules need to know the tag of the container image that should be deployed.
In the previous example, this was set to `@forge(name="GIT_COMMIT_HASH")` which means the tag will always be set to the current
git commit hash.
This is the standard approach as most projects are configured with a [docker release](./releases/docker.md) that is set to
publish images using the git commit hash.

There is no enforced schema for the `values` field as it depends on the module being consumed.
Refer to the documentation for a specific module to determine what fields are available for configuration.

## Templating

!!! note
    Most modules are published to a private AWS ECR instance.
    Because of this, you must be authenticated with AWS before trying to generate a deployment template.
    This necessarily means external contributors will not be able to use this feature.

When the deployment step in the CI activates, Forge automatically compiles the raw Kubernetes manifests from the Timoni bundle and
passes it to Argo CD for reconciliation.
It's possible to generate these manifests yourself using the `forge deploy template` command:

```
forge deploy template <path/to/project>
```

Note that you _must_ have the [Timoni CLI](https://timoni.sh/install/) installed locally for this command to work.
Forge will automatically generate the Timoni bundle and then call the Timoni CLI to convert it to its raw YAML counterpart.
The resulting manifests will then be printed to `stdout`.
This is useful for troubleshooting a deployment as it allows you to examine exactly what is getting deployed to Kubernetes.