package github

type GithubCmd struct {
	Create CreateCmd `cmd:"" help:"Create a new GHA authentication entry."`
	Get    GetCmd    `cmd:"" help:"Get a GHA authentication entry."`
	Update UpdateCmd `cmd:"" help:"Update a GHA authentication entry."`
	Delete DeleteCmd `cmd:"" help:"Delete a GHA authentication entry."`
	List   ListCmd   `cmd:"" help:"List all GHA authentication entries."`
}
