package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"cuelang.org/go/cue/cuecontext"
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/cmd/cmds"
)

var version = "dev"

var cli struct {
	Dump     cmds.DumpCmd     `cmd:"" help:"Dumps a project's blueprint to JSON."`
	Run      cmds.RunCmd      `cmd:"" help:"Run an Earthly target."`
	Scan     cmds.ScanCmd     `cmd:"" help:"Scan for Earthfiles."`
	Secret   cmds.SecretCmd   `cmd:"" help:"Manage secrets."`
	Validate cmds.ValidateCmd `cmd:"" help:"Validates a project."`
	Version  VersionCmd       `cmd:"" help:"Print the version."`
	Verbose  int              `short:"v" type:"counter" help:"Enable verbose logging."`
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
	ctx.Bind(logger)

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
