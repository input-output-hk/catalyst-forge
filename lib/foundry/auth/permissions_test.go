package auth

import "testing"

func TestMatchesDomainPattern(t *testing.T) {
	testCases := []struct {
		name    string
		san     string
		pattern string
		expect  bool
	}{
		// Exact matches
		{"exact match domain", "example.com", "example.com", true},
		{"exact match subdomain", "api.example.com", "api.example.com", true},

		// Admin wildcard
		{"admin wildcard domain", "anything.com", "*", true},
		{"admin wildcard subdomain", "sub.domain.com", "*", true},

		// Subdomain wildcard matches
		{"wildcard subdomain match", "api.example.com", "*.example.com", true},
		{"wildcard deep subdomain match", "sub.api.example.com", "*.example.com", true},
		{"wildcard very deep subdomain", "deep.sub.example.com", "*.example.com", true},

		// Subdomain wildcard non-matches
		{"wildcard root domain exclusion", "example.com", "*.example.com", false},
		{"wildcard different domain", "different.com", "*.example.com", false},
		{"wildcard different subdomain", "api.different.com", "*.example.com", false},

		// Non-matches
		{"different domain", "different.com", "example.com", false},
		{"different subdomain", "api.different.com", "api.example.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchesDomainPattern(tc.san, tc.pattern)
			if result != tc.expect {
				t.Errorf("MatchesDomainPattern(%q, %q) = %t, expected %t",
					tc.san, tc.pattern, result, tc.expect)
			}
		})
	}
}

func TestIsCertificateSignPermission(t *testing.T) {
	testCases := []struct {
		name       string
		permission Permission
		expect     bool
	}{
		{"admin cert permission", "certificate:sign:*", true},
		{"wildcard cert permission", "certificate:sign:*.example.com", true},
		{"specific cert permission", "certificate:sign:api.example.com", true},
		{"cert revoke permission", "certificate:revoke", false},
		{"user read permission", "user:read", false},
		{"empty permission", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCertificateSignPermission(tc.permission)
			if result != tc.expect {
				t.Errorf("IsCertificateSignPermission(%q) = %t, expected %t",
					tc.permission, result, tc.expect)
			}
		})
	}
}

func TestParseCertificateSignPermission(t *testing.T) {
	testCases := []struct {
		name          string
		permission    Permission
		expectOk      bool
		expectPattern string
	}{
		{"admin permission", "certificate:sign:*", true, "*"},
		{"wildcard permission", "certificate:sign:*.example.com", true, "*.example.com"},
		{"specific permission", "certificate:sign:api.example.com", true, "api.example.com"},
		{"revoke permission", "certificate:revoke", false, ""},
		{"user permission", "user:read", false, ""},
		{"empty permission", "", false, ""},
		{"incomplete cert permission", "certificate:sign:", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern, ok := ParseCertificateSignPermission(tc.permission)
			if ok != tc.expectOk {
				t.Errorf("ParseCertificateSignPermission(%q) ok = %t, expected %t",
					tc.permission, ok, tc.expectOk)
			}
			if pattern != tc.expectPattern {
				t.Errorf("ParseCertificateSignPermission(%q) pattern = %q, expected %q",
					tc.permission, pattern, tc.expectPattern)
			}
		})
	}
}

func TestCreateCertificateSignPermission(t *testing.T) {
	testCases := []struct {
		name       string
		pattern    string
		expectPerm Permission
	}{
		{"admin pattern", "*", "certificate:sign:*"},
		{"wildcard pattern", "*.example.com", "certificate:sign:*.example.com"},
		{"specific pattern", "api.example.com", "certificate:sign:api.example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CreateCertificateSignPermission(tc.pattern)
			if result != tc.expectPerm {
				t.Errorf("CreateCertificateSignPermission(%q) = %q, expected %q",
					tc.pattern, result, tc.expectPerm)
			}
		})
	}
}
