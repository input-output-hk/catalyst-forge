package validate

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	cue "cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	ociv2 "github.com/input-output-hk/catalyst-forge/lib/ociv2"
)

//go:embed *.cue
var cueFS embed.FS

type cueJSONValidator struct {
	top string
}

func (v cueJSONValidator) Validate(ctx context.Context, doc []byte) error {
	// Load and combine schema files
	files := []string{"common.cue", "kcl_compat.cue", "kcl_strict.cue"}
	combined := ""
	for _, f := range files {
		b, err := cueFS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read schema %s: %w", f, err)
		}
		combined += string(b) + "\n"
	}

	ctxc := cuecontext.New()
	schema := ctxc.CompileString(combined)
	if err := schema.Err(); err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	topVal := schema.LookupPath(cue.ParsePath(v.top))
	if err := topVal.Err(); err != nil {
		return fmt.Errorf("lookup top %q: %w", v.top, err)
	}

	var data interface{}
	if err := json.Unmarshal(doc, &data); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	dv := ctxc.Encode(data)
	if err := dv.Err(); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}

	unified := topVal.Unify(dv)
	if err := unified.Err(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// NewCompatValidator returns an ociv2.JSONValidator for KCL compat validation
func NewCompatValidator() ociv2.JSONValidator {
	return cueJSONValidator{top: "#KCLCompatValidation"}
}

// NewStrictMetaValidator validates strict meta.json
func NewStrictMetaValidator() ociv2.JSONValidator {
	return cueJSONValidator{top: "#KCLModuleMetaV1"}
}

// NewStrictInputValidator validates strict manifest+meta+tarHex input
func NewStrictInputValidator() ociv2.JSONValidator {
	return cueJSONValidator{top: "#KCLStrictValidation"}
}
