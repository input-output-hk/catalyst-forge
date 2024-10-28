---
draft: false
date: 2024-10-25
authors:
  - jmgilman
---

# What's New in Forge - 10-25-2024

Check out what's new in Forge this week.

<!-- more -->

## Releases

The `publish` and `release` targets are no more!
They have been replaced with an entirely new system that will enable adding more release automation going forward.
Individual releases are now defined in a project's blueprint and Forge will automatically discover and execute them in the CI
pipeline.
Each release is run in parallel to maximize speed.

The old targets will no longer automatically run in the CI.
You will need to configure new releases in your project's blueprint file to continue publishing/releasing.
The `publish` target has been replaced with the `docker` release type.
The `release` target has been replaced with the `github` release type.

For example, you can continue to use the `publish` target in your `Earthfile` by configuring a `docker` release type:

```cue
project: {
    name: "myproject"
    release: {
		docker: {
			on: {
				merge: {}
				tag: {}
			}
			config: {
				tag: _ @forge(name="GIT_COMMIT_HASH")
			}
            target: "publish"
		}
	}
}
```

The above configuration will create a new docker release whenever a merge to the default branch occurs or when a new git tag is
created.
The published image will have its tag (`config.tag` above) automatically filled in with the git commit hash.
Finally, Forge will call the `publish` target in your `Earthfile` to generate the container image.

To learn more about releases, please refer to the [reference documentation](../../reference/releases/index.md).

## Deployment Templates

A new command has been introduced to the CLI: `forge deploy template`.
This command can be used to generate the raw Kubernetes manifests (in YAML) that will be applied to the target Kubernetes cluster
during automatic deployments.
This is useful when troubleshooting why a deplyoment may be failing or acting in an unexpected way.
All generated manifests will be printed to `stdout` and can be redirected to a local file for easier viewing.

The below example shows what it looks like to generate the raw manifests for the Foundry API server:

```text
$ forge deploy template foundry/api
---
# Instance: foundry-api
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/managed-by: timoni
    app.kubernetes.io/name: foundry-api
    app.kubernetes.io/version: 0.1.0
  name: foundry-api
  namespace: default
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  selector:
    app.kubernetes.io/name: foundry-api
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: timoni
    app.kubernetes.io/name: foundry-api
    app.kubernetes.io/version: 0.1.0
  name: foundry-api
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: foundry-api
  template:
    metadata:
      labels:
        app.kubernetes.io/name: foundry-api
    spec:
      containers:
      - image: 332405224602.dkr.ecr.eu-central-1.amazonaws.com/foundry-api:763fe7fd2bfdd39d630df9b5c5aa7e6588fc6eea
        imagePullPolicy: IfNotPresent
        livenessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
        name: foundry-api
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        readinessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
```

For more information, please refer to the [deployments documentation](../../reference/deployments.md#templating).

## What's Next

Work is currenetly being done to improve automatic deployments for projects.
Currently, Forge assumes a GitOps repository exists and will automatically generate and commit updated deployments to the configured
repository.
This makes setup complicated and introduces a mostly unecessary step in the deployment process.

Instead, we are investigating having a GitOps operator (currently only Argo CD) point directly at a project's repository.
Since a blueprint file is self-contained, it's possible to generate Kubernetes manifests using only the information inside of it.
The first steps towards experimenting with this new solution was to create a
[custom management plugin](https://github.com/input-output-hk/catalyst-forge/tree/master/tools/argocd) capable of ingesting a
project and spitting out raw Kubernetes manifests.
With this in place, it should be possible to point Argo CD directly at a project folder and have it generate the necessary manifests
for deploying the project.
As this process matures, more documentation will be released with the updated deployment process.

That's it for this week, thanks for tuning in!