package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDomain(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "normal email",
			email:    "user@example.com",
			expected: "example.com",
		},
		{
			name:     "uppercase domain",
			email:    "user@EXAMPLE.COM",
			expected: "example.com",
		},
		{
			name:     "mixed case domain",
			email:    "user@ExAmPlE.CoM",
			expected: "example.com",
		},
		{
			name:     "email with spaces",
			email:    "  user@example.com  ",
			expected: "example.com",
		},
		{
			name:     "domain with spaces",
			email:    "user@  example.com  ",
			expected: "example.com",
		},
		{
			name:     "subdomain",
			email:    "user@mail.example.com",
			expected: "mail.example.com",
		},
		{
			name:     "no @ symbol",
			email:    "notanemail",
			expected: "unknown",
		},
		{
			name:     "empty string",
			email:    "",
			expected: "unknown",
		},
		{
			name:     "@ at end",
			email:    "user@",
			expected: "unknown",
		},
		{
			name:     "just @",
			email:    "@",
			expected: "unknown",
		},
		{
			name:     "multiple @ symbols",
			email:    "user@subdomain@example.com",
			expected: "example.com",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDomain(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToHex(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "single byte",
			input:    []byte{0x42},
			expected: "42",
		},
		{
			name:     "multiple bytes",
			input:    []byte{0xde, 0xad, 0xbe, 0xef},
			expected: "deadbeef",
		},
		{
			name:     "all zeros",
			input:    []byte{0x00, 0x00},
			expected: "0000",
		},
		{
			name:     "all ones",
			input:    []byte{0xff, 0xff},
			expected: "ffff",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toHex(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		sensitive bool
	}{
		// Sensitive fields
		{"password", "password", true},
		{"PASSWORD", "PASSWORD", true},
		{"Password", "Password", true},
		{"user_password", "user_password", true},
		{"token", "token", true},
		{"access_token", "access_token", true},
		{"refresh_token", "refresh_token", true},
		{"secret", "secret", true},
		{"api_secret", "api_secret", true},
		{"private_key", "private_key", true},
		{"privateKey", "privateKey", true},
		{"credential", "credential", true},
		{"credentialId", "credentialId", true},
		{"signature", "signature", true},
		{"clientDataJSON", "clientDataJSON", true},
		{"authenticatorData", "authenticatorData", true},
		{"publicKey", "publicKey", true},
		{"recovery_code", "recovery_code", true},
		{"hash", "hash", true},
		{"token_hash", "token_hash", true},
		
		// Non-sensitive fields
		{"email", "email", false},
		{"username", "username", false},
		{"user_id", "user_id", false},
		{"roles", "roles", false},
		{"permissions", "permissions", false},
		{"created_at", "created_at", false},
		{"status", "status", false},
		{"message", "message", false},
		{"error", "error", false},
		{"device_name", "device_name", false},
		
		// Edge cases
		{"", "", false},
		{"  ", "  ", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveField(tt.field)
			assert.Equal(t, tt.sensitive, result, "Field %s sensitivity mismatch", tt.field)
		})
	}
}