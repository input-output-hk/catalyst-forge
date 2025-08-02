package auth

// Permission represents a specific action that can be performed
type Permission string

const (
	PermAliasRead            Permission = "alias:read"
	PermAliasWrite           Permission = "alias:write"
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

// AllPermissions is a list of all possible permissions
var AllPermissions = []Permission{
	PermAliasRead,
	PermAliasWrite,
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
