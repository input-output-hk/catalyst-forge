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
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

var version = "dev"

var cli struct {
	cmds.GlobalArgs

	Deploy   cmds.DeployCmd   `kong:"cmd" help:"Deploy a project." `
	Dump     cmds.DumpCmd     `kong:"cmd" help:"Dumps a project's blueprint to JSON."`
	CI       cmds.CICmd       `kong:"cmd" help:"Simulate a CI run."`
	Run      cmds.RunCmd      `kong:"cmd" help:"Run an Earthly target."`
	Scan     cmds.ScanCmd     `kong:"cmd" help:"Scan for Earthfiles."`
	Secret   cmds.SecretCmd   `kong:"cmd" help:"Manage secrets."`
	Tag      cmds.TagCmd      `kong:"cmd" help:"Generate a tag for a project."`
	Validate cmds.ValidateCmd `kong:"cmd" help:"Validates a project."`
	Version  VersionCmd       `kong:"cmd" help:"Print the version."`

	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"install shell completions"`
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
	cliArgs := os.Args[1:]

	parser := kong.Must(&cli,
		kong.Name("forge"),
		kong.Description("The CLI tool powering Catalyst Forge"))

	kongplete.Complete(parser,
		kongplete.WithPredictor("path", complete.PredictFiles("*")),
	)

	ctx, err := parser.Parse(cliArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v\n", err)
		return 1
	}

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

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(Run())
}
