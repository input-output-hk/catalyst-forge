# Blueprint Schema

This directory contains the schema for the blueprint file.
The schema is created from a combination of generated and static code.
The "base" schema is defined as a series of Go structures located in [schema.go](./schema.go).
Schema properties that cannot be expressed via Go structure tags are contained in [schema_overrides.cue](./schema_overrides.cue).

If you're looking for an authoritative source for the schema, see [_embed/schema.cue](./_embed/schema.cue).

## Generation

Generating the schema can be accomplished by running `go generate`.
This causes the following:

1. The Go structures will have their respective CUE definitions generated in `schema_go_gen.cue`
2. All CUE files (including the generated file from the previous step) are consolidated to a single file in `_embed/schema.cue`

The final generated file is embedded into the `SchemaFile` package variable.