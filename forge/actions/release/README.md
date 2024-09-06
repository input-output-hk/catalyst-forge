# Release Action

The release action creates a new GitHub release and uploads artifacts to it.
It automatically handles parsing git tags in order to generate the release name.

## Usage

```yaml
name: Run Release
on:
  push:

permissions:
  contents: read
  id-token: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Setup
        uses: input-output-hk/catalyst-forge/forge/actions/setup@master
      - name: Release
        if: startsWith(github.ref, 'refs/tags/')
        uses: input-output-hk/catalyst-forge/forge/actions/release@master
        with:
          project: ./my/project/path
          path: ./path/to/artifacts
```

The action should only be run when a git tag is present.
The given project is used to determine whether a release should happen or not:

- If the git tag is a mono-repo tag and it matches the given project, then a release is made
- If the git tag is not a mono-repo tag, a release always occurs

The release is named the same as the git tag.
The given `path` is archived in a `.tar.gz` file and uploaded as an asset for the release.
The name of the archive depends on the git tag:

- If the git tag is a mono-repo tag, the archive is named in the format of: `<prefix>-<platform>.tar.gz`
- If the git tag is not a mono-repo tag, the archive is named in the format of: `<repo_name>-<platform>.tar.gz`

## Inputs

| Name    | Description                                        | Required | Default |
| ------- | -------------------------------------------------- | -------- | ------- |
| project | The relative path to the project (from git root)   | Yes      | N/A     |
| path    | The path to any artifacts to attach to the release | Yes      | N/A     |
