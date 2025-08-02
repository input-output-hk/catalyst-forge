package roles

type Cmd struct {
	Assign AssignCmd `kong:"cmd,help='Assign a user to a role'"`
	Remove RemoveCmd `kong:"cmd,help='Remove a user from a role'"`
	List   ListCmd   `kong:"cmd,help='List user-role relationships'"`
}
