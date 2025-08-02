package users

import (
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api/auth/users/roles"
)

type UserCmd struct {
	Create     CreateCmd     `cmd:"" help:"Create a new user."`
	Get        GetCmd        `cmd:"" help:"Get a user by ID or email."`
	Update     UpdateCmd     `cmd:"" help:"Update a user."`
	Delete     DeleteCmd     `cmd:"" help:"Delete a user."`
	List       ListCmd       `cmd:"" help:"List all users."`
	Pending    PendingCmd    `cmd:"" help:"List all users with pending status."`
	Activate   ActivateCmd   `cmd:"" help:"Activate a user by ID or email."`
	Deactivate DeactivateCmd `cmd:"" help:"Deactivate a user."`
	Roles      roles.Cmd     `cmd:"" help:"Manage user roles."`
}
