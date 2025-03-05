# Project

The `project` package provides the Go API code for loading projects from a given directory.
The package also embeds the full schema for blueprint files.

## Usage

### Loading Projects

The `ProjectLoader` can be used to load a project from a given path.
The following is an example that uses the loader to load a project:

```go
package main

import (
	"log"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

func main() {
	loader := project.NewDefaultProjectLoader(
		project.GetDefaultRuntimes(nil),
		nil,
	)

	project, err := loader.Load("/path/to/load")
	if err != nil {
		log.Fatalf("failed to decode blueprint: %v", err)
	}

	log.Printf("Project name: %s", project.Name)
}
```

### Blueprint Schema

The blueprint schema is embedded in the `schema` package and can be loaded using the included function:

```go
package main

import (
	"fmt"
	"log"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
)

func main() {
	ctx := cuecontext.New()
	schema, err := schema.LoadSchema(ctx)
	if err != nil {
		log.Fatalf("failed to load schema: %v", err)
	}

	fmt.Printf("Schema version: %s\n", schema.Version)

	v := schema.Unify(ctx.CompileString(`{version: "1.0"}`))
	if v.Err() != nil {
		log.Fatalf("failed to unify schema: %v", v.Err())
	}
}
```

All blueprints must specify the schema version they are using in the top-level `schema` field.
The schema itself carries its version at `schema.Version`.
This value is managed by Catalyst Forge developers and will periodically change as the schema evolves.
The loader will automatically perform version checks to ensure any parsed blueprints are compatible with the embedded schema.

For more information on the schema, see the [schema README](./schema/README.md).

## Testing

Tests can be run with:

```
go test ./...
```
