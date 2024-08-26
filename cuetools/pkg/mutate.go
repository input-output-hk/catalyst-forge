package pkg

import (
	"fmt"

	"cuelang.org/go/cue"
)

// delete deletes the field at the given path from the given value.
// The path must point to either a struct field or a list index.
func Delete(ctx *cue.Context, v cue.Value, path string) (cue.Value, error) {
	// final holds the final value after the delete operation
	var final cue.Value

	refPath := cue.ParsePath(path)
	refSels := refPath.Selectors()

	// Make sure the target path exists
	if !v.LookupPath(refPath).Exists() {
		return v, fmt.Errorf("path %q does not exist", path)
	}

	// Isolate the last selector in the target path, which is the field to delete
	deletedSel, parentSels := refSels[len(refSels)-1], refSels[:len(refSels)-1]
	parentPath := cue.MakePath(parentSels...) // Path to the parent of the field to delete

	var err error
	final, err = deleteFrom(ctx, v.LookupPath(parentPath), deletedSel)
	if err != nil {
		return v, fmt.Errorf("failed to delete field: %v", err)
	}

	// Replace the parent struct in the given value with the new struct that has the target field removed
	final, err = Replace(ctx, v, parentPath.String(), final)
	if err != nil {
		return v, fmt.Errorf("failed to rebuild struct: %v", err)
	}

	return final, nil
}

// replace replaces the value at the given path with the given value.
// The path must point to either a struct field or a list index.
func Replace(ctx *cue.Context, v cue.Value, path string, replace cue.Value) (cue.Value, error) {
	cpath := cue.ParsePath(path)
	if !v.LookupPath(cpath).Exists() {
		return v, fmt.Errorf("path %q does not exist", path)
	}

	final := replace
	sels := cpath.Selectors()
	for len(sels) > 0 {
		var lastSel cue.Selector
		curIndex := len(sels) - 1
		lastSel, sels = sels[curIndex], sels[:curIndex]

		switch lastSel.Type() {
		case cue.IndexLabel:
			new := ctx.CompileString("[...]")
			curList, err := v.LookupPath(cue.MakePath(sels...)).List()
			if err != nil {
				return cue.Value{}, fmt.Errorf("expected list at path %s: %v", path, err)
			}

			for i := 0; curList.Next(); i++ {
				var val cue.Value
				if curList.Selector() == lastSel {
					val = final
				} else {
					val = curList.Value()
				}

				new = new.FillPath(cue.MakePath(cue.Index(i)), val)
			}

			final = new
		case cue.StringLabel:
			new := ctx.CompileString("{}")
			curFields, err := v.LookupPath(cue.MakePath(sels...)).Fields()
			if err != nil {
				return cue.Value{}, fmt.Errorf("expected struct at path %s: %v", path, err)
			}

			for curFields.Next() {
				fieldPath := cue.MakePath(curFields.Selector())
				if curFields.Selector() == lastSel {
					new = new.FillPath(fieldPath, final)
				} else {
					new = new.FillPath(fieldPath, curFields.Value())
				}
			}

			final = new
		default:
			return cue.Value{}, fmt.Errorf("unknown selector type %s", lastSel.Type())
		}
	}

	return final, nil
}

// deleteFrom deletes the field at the given selector from the given value.
// The value must be a struct or a list.
func deleteFrom(ctx *cue.Context, v cue.Value, targetSel cue.Selector) (cue.Value, error) {
	switch targetSel.Type() {
	case cue.IndexLabel:
		new := ctx.CompileString("[...]")
		list, err := v.List()
		if err != nil {
			return v, fmt.Errorf("expected list: %v", err)
		}

		var i int
		for list.Next() {
			if list.Selector() == targetSel {
				continue
			}

			new = new.FillPath(cue.MakePath(cue.Index(i)), list.Value())
			i++
		}

		return new, nil
	case cue.StringLabel:
		new := ctx.CompileString("{}")
		fields, err := v.Fields()
		if err != nil {
			return v, fmt.Errorf("expected struct: %v", err)
		}

		for fields.Next() {
			if fields.Selector() == targetSel {
				continue
			}

			new = new.FillPath(cue.MakePath(fields.Selector()), fields.Value())
		}

		return new, nil
	default:
		return v, fmt.Errorf("unsupported selector type %s", targetSel.Type().String())
	}
}
