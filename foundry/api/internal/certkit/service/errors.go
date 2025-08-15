package service

import "errors"

var (
	ErrCSRInvalid         = errors.New("csr invalid")
	ErrCSRSignature       = errors.New("csr signature invalid")
	ErrCSRPolicyViolation = errors.New("csr policy violation")
	ErrIssuanceTimeout    = errors.New("issuance timed out")
)
