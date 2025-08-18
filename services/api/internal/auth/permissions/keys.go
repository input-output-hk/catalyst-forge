package permissions

import "sort"

// Exported permission key constants for code references
const (
	// Releases
	ReleaseRead   = "release:read"
	ReleaseCreate = "release:create"
	ReleaseUpdate = "release:update"
	ReleaseDelete = "release:delete"

	// Deployments
	DeployRead    = "deploy:read"
	DeployCreate  = "deploy:create"
	DeployCancel  = "deploy:cancel"
	DeployPromote = "deploy:promote"

	// Projects & Environments
	ProjectRead   = "project:read"
	ProjectUpdate = "project:update"
	EnvRead       = "env:read"
	EnvUpdate     = "env:update"

	// Builds / Artifacts
	BuildTrigger  = "build:trigger"
	BuildRead     = "build:read"
	ArtifactWrite = "artifact:write"
	ArtifactRead  = "artifact:read"

	// RBAC admin
	RBACAdmin = "rbac:admin"

	// Certificates
	CertSign = "cert:sign"

	// Auth policy management
	AuthPoliciesRead  = "auth.policies:read"
	AuthPoliciesWrite = "auth.policies:write"

	// Audit
	AuditRead = "audit:read"

	// Admin: users
	UserRead  = "user:read"
	UserWrite = "user:write"

	// Admin: access requests
	AccessRequestsRead  = "access-requests:read"
	AccessRequestsWrite = "access-requests:write"

	// Admin: invites
	InviteCreate = "invite:create"
)

type Info struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
}

func Catalog() []Info {
	perms := []Info{
		{Key: ReleaseRead, Name: "Read Releases", Description: "View release records and details", Domain: "releases"},
		{Key: ReleaseCreate, Name: "Create Releases", Description: "Create new release records", Domain: "releases"},
		{Key: ReleaseUpdate, Name: "Update Releases", Description: "Modify existing release records", Domain: "releases"},
		{Key: ReleaseDelete, Name: "Delete Releases", Description: "Delete release records", Domain: "releases"},

		{Key: DeployRead, Name: "Read Deployments", Description: "View deployments and status", Domain: "deployments"},
		{Key: DeployCreate, Name: "Create Deployments", Description: "Create or trigger deployments", Domain: "deployments"},
		{Key: DeployCancel, Name: "Cancel Deployments", Description: "Cancel in-progress deployments", Domain: "deployments"},
		{Key: DeployPromote, Name: "Promote Deployments", Description: "Promote deployments between environments", Domain: "deployments"},

		{Key: ProjectRead, Name: "Read Projects", Description: "View project information", Domain: "projects"},
		{Key: ProjectUpdate, Name: "Update Projects", Description: "Modify project settings", Domain: "projects"},
		{Key: EnvRead, Name: "Read Environments", Description: "View environments and configs", Domain: "environments"},
		{Key: EnvUpdate, Name: "Update Environments", Description: "Modify environment configs", Domain: "environments"},

		{Key: BuildTrigger, Name: "Trigger Builds", Description: "Trigger CI builds or jobs", Domain: "builds"},
		{Key: BuildRead, Name: "Read Builds", Description: "View build history and logs", Domain: "builds"},
		{Key: ArtifactWrite, Name: "Write Artifacts", Description: "Publish or attach artifacts", Domain: "artifacts"},
		{Key: ArtifactRead, Name: "Read Artifacts", Description: "View or download artifacts", Domain: "artifacts"},

		{Key: RBACAdmin, Name: "RBAC Admin", Description: "Manage roles, bindings, and RBAC evaluation", Domain: "rbac"},

		{Key: CertSign, Name: "Sign Certificates", Description: "Request certificate issuance (PKI)", Domain: "certificates"},

		{Key: AuthPoliciesRead, Name: "Read Auth Policies", Description: "View authentication/authorization policies", Domain: "auth.policies"},
		{Key: AuthPoliciesWrite, Name: "Write Auth Policies", Description: "Create, update, or delete policies", Domain: "auth.policies"},

		{Key: AuditRead, Name: "Read Audit Logs", Description: "View audit events and trails", Domain: "audit"},

		{Key: UserRead, Name: "Read Users", Description: "List and view user accounts", Domain: "admin.users"},
		{Key: UserWrite, Name: "Write Users", Description: "Modify or delete user accounts", Domain: "admin.users"},

		{Key: AccessRequestsRead, Name: "Read Access Requests", Description: "List and view access requests", Domain: "admin.access-requests"},
		{Key: AccessRequestsWrite, Name: "Write Access Requests", Description: "Approve or reject access requests", Domain: "admin.access-requests"},

		{Key: InviteCreate, Name: "Create Invites", Description: "Issue new invites for onboarding", Domain: "admin.invites"},
	}
	sort.Slice(perms, func(i, j int) bool { return perms[i].Key < perms[j].Key })
	return perms
}

func All() []string {
	infos := Catalog()
	keys := make([]string, 0, len(infos))
	for _, info := range infos {
		keys = append(keys, info.Key)
	}
	return keys
}
