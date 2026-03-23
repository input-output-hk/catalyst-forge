package auth

import "strings"

// Permission represents a specific action that can be performed
type Permission string

const (
	PermAliasRead            Permission = "alias:read"
	PermAliasWrite           Permission = "alias:write"
	PermCertificateRevoke    Permission = "certificate:revoke"
	PermCertificateSignAll   Permission = "certificate:sign:*"
	PermDeploymentRead       Permission = "deployment:read"
	PermDeploymentWrite      Permission = "deployment:write"
	PermDeploymentEventRead  Permission = "deployment:event:read"
	PermDeploymentEventWrite Permission = "deployment:event:write"
	PermReleaseRead          Permission = "release:read"
	PermReleaseWrite         Permission = "release:write"
	PermGHAAuthRead          Permission = "gha:auth:read"
	PermGHAAuthWrite         Permission = "gha:auth:write"
	PermUserRead             Permission = "user:read"
	PermUserWrite            Permission = "user:write"
	PermRoleRead             Permission = "role:read"
	PermRoleWrite            Permission = "role:write"
	PermUserKeyRead          Permission = "user:key:read"
	PermUserKeyWrite         Permission = "user:key:write"
)

// AllPermissions is a list of all possible static permissions
// Note: certificate:sign permissions are dynamic and not included here
var AllPermissions = []Permission{
	PermAliasRead,
	PermAliasWrite,
	PermCertificateRevoke,
	PermCertificateSignAll,
	PermDeploymentRead,
	PermDeploymentWrite,
	PermDeploymentEventRead,
	PermDeploymentEventWrite,
	PermReleaseRead,
	PermReleaseWrite,
	PermGHAAuthRead,
	PermGHAAuthWrite,
	PermUserRead,
	PermUserWrite,
	PermRoleRead,
	PermRoleWrite,
	PermUserKeyRead,
	PermUserKeyWrite,
}

// IsCertificateSignPermission checks if a permission is for certificate signing
func IsCertificateSignPermission(perm Permission) bool {
	return strings.HasPrefix(string(perm), "certificate:sign:")
}

// ParseCertificateSignPermission extracts the domain pattern from a certificate signing permission
// Returns the domain pattern and true if valid, empty string and false if not a certificate permission
func ParseCertificateSignPermission(perm Permission) (string, bool) {
	permStr := string(perm)
	if !strings.HasPrefix(permStr, "certificate:sign:") {
		return "", false
	}

	pattern := permStr[len("certificate:sign:"):]
	if pattern == "" {
		return "", false
	}

	return pattern, true
}

// CreateCertificateSignPermission creates a certificate signing permission for the given domain pattern
func CreateCertificateSignPermission(domainPattern string) Permission {
	return Permission("certificate:sign:" + domainPattern)
}

// MatchesDomainPattern checks if a SAN matches the permission domain pattern
func MatchesDomainPattern(san, pattern string) bool {
	// Exact match
	if san == pattern {
		return true
	}

	// Admin wildcard: * matches everything
	if pattern == "*" {
		return true
	}

	// Wildcard match: *.example.com matches api.example.com but not example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]                                      // Remove * to get .example.com
		return strings.HasSuffix(san, suffix) && san != suffix[1:] // Don't match the root domain
	}

	return false
}
