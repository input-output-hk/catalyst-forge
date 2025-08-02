package auth

import (
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/gha"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/keys"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/roles"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/users"
)

type AuthCmd struct {
	GHA   gha.GhaCmd    `cmd:"" help:"Manage GitHub Actions authentication."`
	Keys  keys.KeysCmd  `cmd:"" help:"Manage user keys."`
	Roles roles.RoleCmd `cmd:"" help:"Manage roles."`
	Users users.UserCmd `cmd:"" help:"Manage users."`
}
