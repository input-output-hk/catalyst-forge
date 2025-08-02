package roles

type RoleCmd struct {
	Create CreateCmd `cmd:"" help:"Create a new role."`
	Get    GetCmd    `cmd:"" help:"Get a role by ID or name."`
	Update UpdateCmd `cmd:"" help:"Update a role."`
	Delete DeleteCmd `cmd:"" help:"Delete a role."`
	List   ListCmd   `cmd:"" help:"List all roles."`
}
