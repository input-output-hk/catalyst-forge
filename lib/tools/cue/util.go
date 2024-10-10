package cue

import "cuelang.org/go/cue"

// FindAttr finds an attribute with the given name in the given CUE value.
func FindAttr(v cue.Value, name string) *cue.Attribute {
	for _, attr := range v.Attributes(cue.FieldAttr) {
		if attr.Name() == name {
			return &attr
		}
	}
	return nil
}
