package module

type ModuleCmd struct {
	Deploy   DeployCmd   `cmd:"" help:"Deploys a module (or project) to the configured GitOps repository."`
	Template TemplateCmd `cmd:"" help:"Generates a module's (or project's) deployment YAML."`
	Values   ValuesCmd   `cmd:"" help:"Gets a module's (or project's) values."`
}
