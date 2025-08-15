package cueval

import (
	"context"
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// Schema represents a compiled CUE schema for validation
type Schema struct {
	top     string    // top-level value name to validate against
	ctx     *cue.Context
	schema  cue.Value // compiled schema
}

// NewFromBytes compiles a CUE schema from bytes and selects the top-level value
func NewFromBytes(schema []byte, top string) (*Schema, error) {
	ctx := cuecontext.New()
	
	// Compile the schema
	schemaValue := ctx.CompileBytes(schema)
	if err := schemaValue.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile CUE schema: %w", err)
	}
	
	// Look up the top-level value
	topValue := schemaValue.LookupPath(cue.ParsePath(top))
	if err := topValue.Err(); err != nil {
		return nil, fmt.Errorf("failed to lookup top-level value %q: %w", top, err)
	}
	
	return &Schema{
		top:    top,
		ctx:    ctx,
		schema: topValue,
	}, nil
}

// NewFromString compiles a CUE schema from a string
func NewFromString(schema string, top string) (*Schema, error) {
	return NewFromBytes([]byte(schema), top)
}

// Validate validates a JSON document against the compiled schema
func (s *Schema) Validate(ctx context.Context, doc []byte) error {
	// Parse the JSON document
	var data interface{}
	if err := json.Unmarshal(doc, &data); err != nil {
		return fmt.Errorf("failed to parse JSON document: %w", err)
	}
	
	// Convert to CUE value
	docValue := s.ctx.Encode(data)
	if err := docValue.Err(); err != nil {
		return fmt.Errorf("failed to encode document as CUE value: %w", err)
	}
	
	// Unify with schema
	unified := s.schema.Unify(docValue)
	if err := unified.Err(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Validate that the unified value is concrete
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	return nil
}

// ValidatorFunc is a helper type that wraps a validation function as a JSONValidator
type ValidatorFunc func(context.Context, []byte) error

// Validate implements the JSONValidator interface
func (f ValidatorFunc) Validate(ctx context.Context, doc []byte) error {
	return f(ctx, doc)
}

// AsValidator returns the schema as a JSONValidator interface
func (s *Schema) AsValidator() ValidatorFunc {
	return ValidatorFunc(s.Validate)
}