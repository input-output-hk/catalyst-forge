# Cue Tools

The `cuetools` package provides common utilities for interacting with the [CUE language](https://cuelang.org/).
The functions contained within this package are used across multiple packages within Catalyst Forge.
However, all functions are self-contained, and they may prove vaulable even outside the context of Catalyst Forge.

## Loading and Validation

The contents of a CUE file can be loaded with:

```go
package pkg

import (
	"fmt"
	"log"
	"os"

	"cuelang.org/go/cue/cuecontext"
    cuetools "github.com/input-output-hk/catalyst-forge/cuetools/pkg"
)

func main() {
	b, err := os.ReadFile("file.cue")
	if err != nil {
		log.Fatal(err)
	}

	ctx := cuecontext.New()
	v, err := cuetools.Compile(ctx, b)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Value: %v\n", v)
}
```

The `Compile` function will only return an error when a syntax error is present in the given file.
It does not validate whether the file is logically valid (i.e., non-concrete values are acceptable).
To further validate the file:

```go
func main() {
    // ...
    err = cuetools.Validate(v, cue.Concrete(true))
    if err != nil {
        log.Fatal(err)
    }
}
```

By default, the validation method provided by the CUE API will ellide error messages when more than one error exists.
The `Validate` method handles this by building a proper error string that includes all errors encountered while validating.
Each error is placed on a new line in order to improve readability.

## Mutating Values

By default, CUE is immutable and it's not possible to arbitrarily delete and/or replace fields within a CUE value.
This constraint exists at the language level and cannot be easily broken via the Go API.
While respecting language boundaries is often the best solution, in some cases it may be overwhelmingly apparent that a field needs
to be mutated and that it can be done safely.
For those cases, this package provides functions for both deleting and replacing arbitrary fields.

To delete a field:

```go
package main

import (
	"fmt"
	"log"

	"cuelang.org/go/cue/cuecontext"
	cuetools "github.com/input-output-hk/catalyst-forge/cuetools/pkg"
)

func main() {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{a: 1, b: 2}`)
	v, err := cuetools.Delete(ctx, v, "a")
	if err != nil {
		log.Fatalf("failed to delete field: %v", err)
	}

	fmt.Println(v) // { b: 2 }
}
```

To replace a field with a new value:

```go
func main() {
    // ...
    v = ctx.CompileString(`{a: 1, b: 2}`)
	v, err := cuetools.Replace(ctx, v, "a", ctx.CompileString("3"))
	if err != nil {
		log.Fatalf("failed to delete field: %v", err)
	}

	fmt.Println(v) // { a: 3, b: 2}
}
```

The `path` argument for both functions can be nested:

```
a.b.c
```

And can also index into lists:

```
a.b[0].c
```

## Testing

Tests can be run with:

```
go test ./...
```