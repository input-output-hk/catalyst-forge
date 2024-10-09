package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"cuelang.org/go/cue/cuecontext"
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

var version = "dev"

var cli struct {
	cmds.GlobalArgs

	Deploy   cmds.DeployCmd   `cmd:"" help:"Deploy a project."`
	Dump     cmds.DumpCmd     `cmd:"" help:"Dumps a project's blueprint to JSON."`
	CI       cmds.CICmd       `cmd:"" help:"Simulate a CI run."`
	Release  cmds.ReleaseCmd  `cmd:"" help:"Release a project."`
	Run      cmds.RunCmd      `cmd:"" help:"Run an Earthly target."`
	Scan     cmds.ScanCmd     `cmd:"" help:"Scan for Earthfiles."`
	Secret   cmds.SecretCmd   `cmd:"" help:"Manage secrets."`
	Tag      cmds.TagCmd      `cmd:"" help:"Generate a tag for a project."`
	Validate cmds.ValidateCmd `cmd:"" help:"Validates a project."`
	Version  VersionCmd       `cmd:"" help:"Print the version."`
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	ctx := cuecontext.New()
	schema, err := schema.LoadSchema(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("forge version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("config schema version %s\n", schema.Version)
	return nil
}

// Run is the entrypoint for the CLI tool.
func Run() int {
	ctx := kong.Parse(&cli,
		kong.Name("forge"),
		kong.Description("The CLI tool powering Catalyst Forge"))

	handler := log.New(os.Stderr)
	switch cli.Verbose {
	case 0:
		handler.SetLevel(log.FatalLevel)
	case 1:
		handler.SetLevel(log.WarnLevel)
	case 2:
		handler.SetLevel(log.InfoLevel)
	case 3:
		handler.SetLevel(log.DebugLevel)
	}

	logger := slog.New(handler)
	loader := project.NewDefaultProjectLoader(logger)
	runctx := run.RunContext{
		CI: cli.GlobalArgs.CI,
		Executor: executor.NewLocalExecutor(
			logger,
			executor.WithRedirect(),
		),
		FSWalker:      walker.NewDefaultFSWalker(logger),
		Local:         cli.GlobalArgs.Local,
		Logger:        logger,
		ProjectLoader: &loader,
		SecretStore:   secrets.NewDefaultSecretStore(),
		Verbose:       cli.GlobalArgs.Verbose,
	}
	ctx.Bind(runctx)

	err := ctx.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(Run())
}
