package blueprint

import "embed"

//go:generate go run cuelang.org/go/cmd/cue@v0.12.0 exp gengotypes

//go:embed cue.mod/module.cue
//go:embed common/*.cue
//go:embed global/*.cue
//go:embed global/providers/*.cue
//go:embed project/*.cue
//go:embed main.cue
var Module embed.FS

// SCHEMA_PACKAGE is the name of the package that contains the blueprint schema.
const SCHEMA_PACKAGE = "blueprint"
