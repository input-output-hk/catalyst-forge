package injector

import (
	"fmt"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

// compileWithConcrete compiles a CUE value for the given attrType and raw string value.
// When concrete is false, it constructs a marked disjunction so the provided value
// is a default (overrideable) rather than a final concrete value.
//
// Examples:
// - string, concrete: "bar"
// - string, default:  *"bar" | string
// - int, concrete:    1
// - int, default:     *1 | int
// - bool, concrete:   true
// - bool, default:    *true | bool
func compileWithConcrete(ctx *cue.Context, attrType AttrType, raw string, concrete bool) (cue.Value, error) {
	switch attrType {
	case AttrTypeString:
		if concrete {
			return ctx.CompileString(fmt.Sprintf("%q", raw)), nil
		}
		return ctx.CompileString(fmt.Sprintf("*%q | string", raw)), nil

	case AttrTypeInt:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return cue.Value{}, fmt.Errorf("invalid int value '%s'", raw)
		}
		if concrete {
			return ctx.CompileString(fmt.Sprintf("%d", n)), nil
		}
		return ctx.CompileString(fmt.Sprintf("*%d | int", n)), nil

	case AttrTypeBool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return cue.Value{}, fmt.Errorf("invalid bool value '%s'", raw)
		}
		lit := "false"
		if b {
			lit = "true"
		}
		if concrete {
			return ctx.CompileString(lit), nil
		}
		return ctx.CompileString(fmt.Sprintf("*%s | bool", lit)), nil

	default:
		return cue.Value{}, fmt.Errorf("unsupported attribute type '%s'", attrType)
	}
}

// makeDefault wraps a cue.Value as a default (marked disjunction) maintaining its type where possible.
// For primitives, it emits a typed default (e.g., *"x" | string). For numbers, it chooses int vs number
// based on the Value's kind. For complex values, it serializes the value syntax and wraps it: *(<v>) | _.
func makeDefault(ctx *cue.Context, v cue.Value) (cue.Value, error) {
	k := v.Kind()
	switch k {
	case cue.StringKind:
		s, err := v.String()
		if err != nil {
			return cue.Value{}, err
		}
		return ctx.CompileString(fmt.Sprintf("*%q | string", s)), nil
	case cue.BoolKind:
		b, err := v.Bool()
		if err != nil {
			return cue.Value{}, err
		}
		lit := "false"
		if b {
			lit = "true"
		}
		return ctx.CompileString(fmt.Sprintf("*%s | bool", lit)), nil
	case cue.IntKind:
		i, err := v.Int64()
		if err != nil {
			return cue.Value{}, err
		}
		return ctx.CompileString(fmt.Sprintf("*%d | int", i)), nil
	case cue.NumberKind:
		f, err := v.Float64()
		if err != nil {
			return cue.Value{}, err
		}
		return ctx.CompileString(fmt.Sprintf("*%g | number", f)), nil
	default:
		// Fallback for structs, lists, etc.: serialize and wrap as default against top
		n := v.Syntax(cue.Concrete(true))
		b, err := format.Node(n)
		if err != nil {
			return cue.Value{}, err
		}
		return ctx.CompileString(fmt.Sprintf("*(%s) | _", string(b))), nil
	}
}
