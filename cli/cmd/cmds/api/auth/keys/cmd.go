package keys

type KeysCmd struct {
	Create   CreateCmd   `cmd:"" help:"Create a new user key."`
	Get      GetCmd      `cmd:"" help:"Get a user key by ID/KID or get keys for a user."`
	Update   UpdateCmd   `cmd:"" help:"Update a user key."`
	Delete   DeleteCmd   `cmd:"" help:"Delete a user key."`
	List     ListCmd     `cmd:"" help:"List all user keys."`
	Pending  PendingCmd  `cmd:"" help:"List all user keys with inactive status for a user."`
	Activate ActivateCmd `cmd:"" help:"Activate a user key by KID with user ID or email."`
	Revoke   RevokeCmd   `cmd:"" help:"Revoke a user key."`
}
