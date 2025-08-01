package auth

import "github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/gha"

type AuthCmd struct {
	GHA gha.GhaCmd `cmd:"" help:"Manage GitHub Actions authentication."`
}
