# Release Action

The publish action pushes Docker images to remote registries using settings from a project's blueprint file.
It automatically handles the configured tagging strategy as well as properly handling git tags.

## Usage

```yaml
name: Run Publish
on:
  push:

permissions:
  contents: read
  id-token: write

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - name: Setup
        uses: input-output-hk/catalyst-forge/forge/actions/setup@master
      - name: Publish
        uses: input-output-hk/catalyst-forge/forge/actions/discover@master
        with:
          project: ./my/project/path
          container: container:tag
```

The given project _must_ have a blueprint at the root of the path which, at the very least, declares a project name.
By default, the `container` property of a project uses the project name if not specified.
Alternatively, the `container` field can be set explicitly.
The `container` input to the publish action _must_ match an existing container image in the local Docker daemon.
The given container name is discarded and the value of the `container` field is used for naming the container.

The final tags the container is published with are determined by the blueprint configuration and the git context:

- The `global.ci.tagging.strategy` configuration property determines the default tag given to all images
- If the git context contains a git tag, then the publish action may or may not publish an image with the tag:
  - If the tag is in the "mono-repo" style (`some/path/v1.0.0`)
    - If the path (`some/path`) matches an alias in `global.ci.tagging.strategy.aliases`, and the value of the alias matches the
      given project, then the tag is used
    - If the path does not match an alias, but the path itself matches the given project, then the tag is used
    - If none of the above are true, the tag is assumed to be for a different project and is skipped
  - If the tag is any other style, it's used as-is (no modifications)

The following table provides an example of how the git tag is used in various contexts:

| Project       | Git tag              | Aliases                  | Image tag |
| ------------- | -------------------- | ------------------------ | --------- |
| `my/cool/cli` | None                 | None                     | Not used  |
| `my/cool/cli` | `v1.0.0`             | None                     | `v1.0.0`  |
| `my/cool/cli` | `my/v1.0.0`          | None                     | Not used  |
| `my/cool/cli` | `my/cool/cli/v1.0.0` | None                     | `v1.0.0`  |
| `my/cool/cli` | `cli/v1.0.0`         | `{"cli": "my/cool/cli"}` | `v1.0.0`  |

After processing any additional tags, the container is retagged with each generated tag and pushed to all registries configured in
`global.ci.registries`.
The publish action assumes the local Docker daemon is already authenticated to any configured registries.

## Inputs

| Name    | Description                                      | Required | Default |
| ------- | ------------------------------------------------ | -------- | ------- |
| project | The relative path to the project (from git root) | Yes      | N/A     |
| image   | The existing container image to publish          | Yes      | N/A     |
