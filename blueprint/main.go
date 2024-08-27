package main

import (
	"fmt"
	"log"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
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
