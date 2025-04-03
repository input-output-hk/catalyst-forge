package deploy

type DeployCmd struct {
	Create DeployCreateCmd `cmd:"create" help:"Create a new deployment."`
	Get    DeployGetCmd    `cmd:"get" help:"Get a deployment."`
}
