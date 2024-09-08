package cue

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
)

// Compile compiles the given CUE contents and returns the resulting value.
// If the contents are invalid, an error is returned.
func Compile(ctx *cue.Context, contents []byte) (cue.Value, error) {
	v := ctx.CompileBytes(contents)
	if v.Err() != nil {
		return cue.Value{}, v.Err()
	}

	return v, nil
}

// Validate validates the given CUE value. If the value is invalid, an error
// is returned.
func Validate(c cue.Value, opts ...cue.Option) error {
	if err := c.Validate(opts...); err != nil {
		var errStr string
		errs := errors.Errors(err)

		if len(errs) == 1 {
			errStr = errs[0].Error()
		} else {
			errStr = "\n"
			for _, e := range errs {
				errStr += e.Error() + "\n"
			}
		}
		return fmt.Errorf("failed to validate: %s", errStr)
	}

	return nil
}
