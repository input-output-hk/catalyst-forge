package internal

import "errors"

// Standard errors used by internal packages
// These mirror the errors in the parent package
var (
	ErrNotFound     = errors.New("oci: not found")
	ErrUnauthorized = errors.New("oci: unauthorized")
	ErrForbidden    = errors.New("oci: forbidden")
	ErrTimeout      = errors.New("oci: timeout")
	ErrCanceled     = errors.New("oci: canceled")
	ErrMediaType    = errors.New("oci: unexpected media type")
	ErrUnsupported  = errors.New("oci: unsupported")
)