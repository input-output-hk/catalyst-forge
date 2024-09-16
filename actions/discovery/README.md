# Discover Action

The discover action acts as a wrapper over the `forge scan` command from the Forge CLI.
It provides inputs that map to their respective flags.
The result from running the command is returned in the `result` output.
By default, the `--enumerate` flag is passed as this is usually the desired output format in CI.

For more information on the `scan` command, refer to the Forge CLI documentation.

## Usage

```yaml
name: Run Setup
on:
  push:

permissions:
  contents: read
  id-token: write

jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - name: Setup
        uses: input-output-hk/catalyst-forge/actions/setup@master
      - name: Discover
        id: discovery
        uses: input-output-hk/catalyst-forge/actions/discover@master
        with:
          filters: |
            ^check.*
            ^test.*
      - name: Show result
        run: echo "${{ steps.discovery.outputs.result }}
```

## Inputs

| Name      | Description                                   | Required | Default   |
| --------- | --------------------------------------------- | -------- | --------- |
| absolute  | Output absolute paths                         | No       | `"false"` |
| enumerate | Enumerate results into Earthfile+Target pairs | No       | `"true"`  |
| filters   | A newline separated list of filters to apply  | No       | `""`      |
| path      | The path to search from                       | No       | `"."`     |
