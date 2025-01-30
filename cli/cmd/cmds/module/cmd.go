package module

type ModuleCmd struct {
	Deploy   DeployCmd   `cmd:"" help:"Deploys a project to the configured GitOps repository."`
	Dump     DumpCmd     `cmd:"" help:"Dumps a project's deployment modules."`
	Template TemplateCmd `cmd:"" help:"Generates a project's (or module's) deployment YAML."`
}
