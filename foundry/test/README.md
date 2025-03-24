# Foundry Test Environment

This directory contains scripts for standing up a full test environment for Foundry using [Kind](https://kind.sigs.k8s.io/).
It includes all necessary dependencies to test the full deployment process using both the Foundry API and Operator.

## Pre-requisites

- [Just](https://github.com/casey/just)
- [Kind](https://kind.sigs.k8s.io/)
- [Earthly](https://earthly.dev/earthfile)
- [jq](https://jqlang.org/)
- [curl](https://curl.se/)

## Usage

To run a completely independent environment, run the following:

```
just up
```

This will perform the following:

1. Create a new Kind cluster along with a shared container registry
2. Deploy an instance of PostgreSQL
3. Deploy an instance of [gitea](https://about.gitea.com/)
4. Build the local version of Foundry API and deploy it
5. Build the local version of Foundry Operator and deploy it (including all supporting resources)
6. Push a [deployment](./repos/deploy/) and a [source](./repos/source/) repo to Gitea

To create a new test release, simply run:

```
just release
```

### Local Usage

The operator is the most complex piece of the Foundry stack and it's often easier to troubleshoot by running it locally.
This can be achieved by instead running:

```
just up-local
```

Which will skip deploying the operator.
After the cluster is up, you can run a local version of the operator by running:

```
just operator-local
```

Then you can create a test deployment with:

```
just release-local
```

Note that by default the operator caches repositories at `$HOME/.cache/forge` which needs to be writable.
You can clean up the cache by running:

```
just cleanup-local
```

## Updating

If you make changes to either the API or operator code, you can quickly deploy the new containers with:

```
# For API
just api

# For operator
just operator
```