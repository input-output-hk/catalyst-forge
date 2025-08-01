# Reject Earthfile Action

This GitHub Action scans for Earthfiles that match certain filter expressions and fails the workflow if any matches are found. This is useful for enforcing policies or preventing certain patterns in Earthfiles.

## Inputs

### `filters` (required)

A newline-separated list of filter expressions and error messages. Each filter should be in the format `"expression, error message"`.

**Example:**

```
prod, contain production targets
debug, contain debug targets
```

### `root-path` (optional)

The root path to scan for Earthfiles. Defaults to the current directory (`.`).

### `filter-source` (optional)

The source to filter by. Can be either `"earthfile"` or `"targets"`. Defaults to `"targets"`.

- `"targets"`: Filter by target names in Earthfiles
- `"earthfile"`: Filter by Earthfile contents

### `verbosity` (optional)

The verbosity level for the forge command. Can be `"error"`, `"info"`, or `"debug"`. Defaults to `"info"`.

## Usage

### Basic Example

```yaml
- name: Reject Earthfiles with production targets
  uses: ./.github/actions/reject-earthfile
  with:
    filters: "prod, contain production targets"
```

### Multiple Filters

```yaml
- name: Reject problematic Earthfiles
  uses: ./.github/actions/reject-earthfile
  with:
    filters: |
      prod, contain production targets
      debug, contain debug targets
      test, contain test targets
```

### Filter by Earthfile Contents

```yaml
- name: Reject Earthfiles with certain content
  uses: ./.github/actions/reject-earthfile
  with:
    filters: "FROM alpine, use Alpine base image"
    filter-source: "earthfile"
```

### Custom Root Path

```yaml
- name: Reject Earthfiles in specific directory
  uses: ./.github/actions/reject-earthfile
  with:
    filters: "prod, contain production targets"
    root-path: "./apps"
```

## Output Format

When rejections are found, the action will output a message in this format:

```
The following Earthfiles contain production targets:
- ./path/to/earthfile1+target1
- ./path/to/earthfile2+target2

The following Earthfiles contain debug targets:
- ./path/to/earthfile3+target3
```

## Filter Expressions

The filter expressions are regular expressions that are matched against either:

- Target names (when `filter-source` is `"targets"`)
- Earthfile contents (when `filter-source` is `"earthfile"`)

### Examples

- `"prod"` - Matches any target or content containing "prod"
- `"^prod"` - Matches targets or content starting with "prod"
- `"prod$"` - Matches targets or content ending with "prod"
- `"prod|staging"` - Matches targets or content containing "prod" or "staging"
