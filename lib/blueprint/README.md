# Blueprint

The `blueprint` package provides the Go API code for loading blueprint files from a given directory.
It provides all necessary functionality to scan, load, and unify one or more blueprint files.
Additionally, the `blueprint` package embeds the full schema for blueprint files.
Every blueprint loaded is automatically validated against the embedded schema.

## Usage

### Loading Blueprint Files

The `BlueprintLoader` can be used to load blueprint files from a given path.
By default, the loader performs the following:

1. Walks the filesystem searching for `blueprint.cue` files
   1. If the path is in a git repository, it walks up to the root of the repository
   2. If the path is not in a git repository, it only searches the given path
2. Loads and processes all found blueprint files (including things like injecting environment variables)
3. Unifies all blueprint files into a single blueprint (including handling versions)
4. Validates the final blueprint against the embedded schema

The loader's `Decode` function can be used to get a `Blueprint` structure that represents the final unified blueprint.
The following is an example that uses the loader to load blueprints:

```go
package main

import (
	"log"

	"github.com/input-output-hk/catalyst-forge/lib/blueprint/pkg/loader"
)

func main() {
	loader := loader.NewDefaultBlueprintLoader("/path/to/load", nil)
	if err := loader.Load(); err != nil {
		log.Fatalf("failed to load blueprint: %v", err)
	}

	bp, err := loader.Decode()
	if err != nil {
		log.Fatalf("failed to decode blueprint: %v", err)
	}

	log.Printf("blueprint: %v", bp)
}
```

If no blueprint files are found, the loader will return a `Blueprint` structure with default values provided for all fields.

### Blueprint Schema

The blueprint schema is embedded in the `schema` package and can be loaded using the included function:

```go
package main

import (
	"fmt"
	"log"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/blueprint/schema"
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
