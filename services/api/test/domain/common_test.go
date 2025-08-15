//go:build integration

package domain

import (
	"testing"
)

// Domain package uses the same TEST_AUTH_BYPASS defaults as others; if endpoints require auth,
// we can add helpers here to attach headers. For now, tests hit public/list endpoints.

func TestDomain_Package_Smoke(t *testing.T) {
	// placeholder to ensure package compiles and TestMain runs
}
