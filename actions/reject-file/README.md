# Reject File Action

This GitHub Action scans for files that match certain filename or content filter expressions and fails the workflow if any matches are found. This is useful for enforcing policies, preventing certain patterns, or ensuring compliance across your codebase.

## Inputs

### `filters` (required)

Filter rules separated by double newlines (`\n\n`). Each rule contains:

- One or more `file:` lines with filename patterns (regex)
- One or more `content:` lines with content patterns (regex)
- Exactly one `description:` line explaining what this rule rejects

Multiple patterns within a rule are ANDed together (all must match).
At least one pattern (`file:` or `content:`) must be specified per rule.

**Format:**

```
file:pattern1
file:pattern2
content:pattern1
description:Error message

file:pattern3
content:pattern2
content:pattern3
description:Another error message
```

**Example:**

```
file:\.key$
description:Private key files should not be committed

content:console\.log
content:console\.error
description:Debug console statements
```

### `root-path` (optional)

The root path to scan for files. Defaults to the repository root (`${{ github.workspace }}`).

### `verbosity` (optional)

The verbosity level for the forge command. Can be `"error"`, `"info"`, or `"debug"`. Defaults to `"info"`.

## Usage

### Basic Example

```yaml
- name: Reject files with secrets
  uses: ./.github/actions/reject-file
  with:
    filters: |
      file:secret
      description:Files containing "secret" in their name
```

### Multiple Rules

```yaml
- name: Reject problematic files
  uses: ./.github/actions/reject-file
  with:
    filters: |
      file:\.key$
      description:Private key files

      content:password\s*=
      description:Hardcoded passwords

      file:\.env$
      description:Environment files that should not be committed
```

### Filter by File Content Only

```yaml
- name: Reject files with TODOs or FIXMEs
  uses: ./.github/actions/reject-file
  with:
    filters: |
      content:TODO|FIXME
      description:Unresolved TODO or FIXME comments
```

### Multiple Patterns in Same Rule

```yaml
- name: Reject test files with debug code
  uses: ./.github/actions/reject-file
  with:
    filters: |
      file:_test\.go$
      content:fmt\.Println
      description:Test files containing debug print statements

      content:console\.log
      content:console\.error
      description:Files with any console logging
```

### Multiple File Patterns

```yaml
- name: Reject various config files
  uses: ./.github/actions/reject-file
  with:
    filters: |
      file:\.env$
      file:\.env\..*
      file:config\.json$
      description:Configuration files that should not be committed
```

### Custom Root Path

```yaml
- name: Reject files in specific directory
  uses: ./.github/actions/reject-file
  with:
    filters: |
      content:console\.log
      description:Console log statements in production code
    root-path: "./src"  # Scan only the src directory instead of entire repo
```

## Output Format

When rejections are found, the action will output a message in this format:

```
Some files failed to pass the filter expressions:

❌ Files containing "secret" in their name:
  - ./config/secret.json
  - ./keys/api-secret.key

❌ Hardcoded passwords:
  - ./src/config.js
  - ./test/fixtures.js
```

## Filter Patterns

### Filename Patterns

Matches against the filename using regular expressions. The pattern is matched against the base filename only (not the full path).

**Examples:**

- `\.key$` - Files ending with .key
- `^test` - Files starting with "test"
- `secret|private` - Files containing "secret" or "private"

### Content Patterns

Matches against the file content using regular expressions. The entire file content is searched.

**Examples:**

- `TODO` - Files containing "TODO"
- `password\s*=` - Files with password assignments
- `console\.log|console\.error` - Files with console statements

### Combined Patterns

When multiple patterns are specified in the same rule, files must match ALL patterns to be rejected. This allows for very specific filtering.

**Example:**

```
file:_test\.js$
content:describe\(
description:Test files that don't use the describe function
```

## Regular Expression Notes

All patterns are treated as regular expressions. Common regex features:

- `^` - Start of string
- `$` - End of string
- `.` - Any character (use `\.` for literal dot)
- `*` - Zero or more of the preceding character
- `+` - One or more of the preceding character
- `|` - OR operator
- `\s` - Whitespace character
- `\w` - Word character (letters, numbers, underscore)
