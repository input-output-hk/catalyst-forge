//go:build integration

package testutil

import (
	"context"
	"fmt"
	"net/http"
)

type BootstrapInvite struct {
	ID    uint   `json:"id"`
	Token string `json:"token"`
}

// BootstrapAdmin calls the new AuthKit bootstrap endpoint to create the initial admin.
// Note: The bootstrap flow may include an access_token; return it when present.
func BootstrapAdmin(ctx context.Context, baseURL, bootstrapToken, email string) (string, error) {
	bootstrapURL := baseURL + "/api/v1/auth/bootstrap"
	var out map[string]any
	if _, err := DoJSON(nil, http.MethodPost, bootstrapURL, nil, map[string]string{
		"email":           email,
		"bootstrap_token": bootstrapToken,
	}, &out); err != nil {
		return "", fmt.Errorf("bootstrap admin failed: %w", err)
	}
	if v, ok := out["access_token"].(string); ok && v != "" {
		return v, nil
	}
	return "", nil
}
