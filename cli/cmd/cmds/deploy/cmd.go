package deploy

type DeployCmd struct {
	Push     PushCmd     `cmd:"" help:"Pushes a project deployment to the GitOps repo."`
	Template TemplateCmd `cmd:"" help:"Generates a project's deployment YAML."`
}
