//go:build integration

package testutil

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CreateBearerForTestSubject builds a bearer token without DB using TEST_AUTH_BYPASS.
// Prefer header-based bypass; this function is provided for cases where a token is required.
func CreateBearerForTestSubject(privateKeyPath, issuer, email string, roles []string) (string, string, error) {
	b, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("read key: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return "", "", fmt.Errorf("invalid pem")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("parse key: %w", err)
	}
	uid := uuid.New()
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":   uid.String(),
		"email": email,
		"roles": roles,
		"iss":   issuer,
		"aud":   issuer,
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
		"sv":    1,
		"jti":   uuid.NewString(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	s, err := tok.SignedString(key)
	if err != nil {
		return "", "", err
	}
	return uid.String(), s, nil
}

// WithTestUserHeaders returns headers for TestAuthBypass: non-admin by default.
func WithTestUserHeaders(ctx context.Context, email string, roles []string, perms []string) map[string]string {
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	uid := uuid.NewString()
	return map[string]string{
		"X-Test-User-ID":     uid,
		"X-Test-Email":       email,
		"X-Test-Roles":       sliceCSV(roles),
		"X-Test-Permissions": sliceCSV(perms),
	}
}

func sliceCSV(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
