package release

type ReleaseCmd struct {
	Create ReleaseCreateCmd `cmd:"create" help:"Create a new release for a project."`
	List   ReleaseListCmd   `cmd:"list" help:"List all releases for a project."`
	Get    ReleaseGetCmd    `cmd:"get" help:"Get a release."`
}
