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
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

var version = "dev"

var cli struct {
	cmds.GlobalArgs

	Deploy   cmds.DeployCmd   `cmd:"" kong:"cmd" help:"Deploy a project." `
	Dump     cmds.DumpCmd     `cmd:"" kong:"cmd" help:"Dumps a project's blueprint to JSON."`
	Devx     cmds.DevX        `cmd:"" kong:"cmd" help:"Reads a forge markdown file and executes a command."`
	CI       cmds.CICmd       `cmd:"" kong:"cmd" help:"Simulate a CI run."`
	Run      cmds.RunCmd      `cmd:"" kong:"cmd" help:"Run an Earthly target."`
	Scan     cmds.ScanCmd     `cmd:"" kong:"cmd" help:"Scan for Earthfiles."`
	Secret   cmds.SecretCmd   `cmd:"" kong:"cmd" help:"Manage secrets."`
	Tag      cmds.TagCmd      `cmd:"" kong:"cmd" help:"Generate a tag for a project."`
	Validate cmds.ValidateCmd `cmd:"" kong:"cmd" help:"Validates a project."`
	Version  VersionCmd       `cmd:"" kong:"cmd" help:"Print the version."`

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
		kongplete.WithPredictor("file", complete.PredictFiles("*")),
		kongplete.WithPredictor("devx-commands", complete.PredictSet("tmp1", "tmp2", "tmp3")),
	)

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

	ctx, err := parser.Parse(cliArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v\n", err)
		return 1
	}

	runctx := run.RunContext{
		CI:      cli.GlobalArgs.CI,
		Local:   cli.GlobalArgs.Local,
		Verbose: cli.GlobalArgs.Verbose,
	}
	ctx.Bind(runctx, slog.New(handler))

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(Run())
}
