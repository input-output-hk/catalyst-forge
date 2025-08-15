package validate

import (
	"context"
)

// Stubbed validators (real CUE-backed implementation can be enabled under a build tag)

func ValidateStrictMeta(metaJSON []byte) error {
	// TODO: replace with CUE-backed validator when module/tooling is wired in
	return nil
}

func ValidateStrictInput(ctx context.Context, doc []byte) error {
	// TODO: replace with CUE-backed validator when module/tooling is wired in
	return nil
}
