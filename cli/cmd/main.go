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
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/api"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/module"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cmds/scan"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/config"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/git"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	schema "github.com/input-output-hk/catalyst-forge/lib/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

var version = "dev"

type GlobalArgs struct {
	ApiURL  string `help:"URL of the Foundry API (overrides the global configuration)."`
	CI      bool   `help:"Run in CI mode."`
	Local   bool   `short:"l" help:"Forces all runs to happen locally (ignores any remote satellites)."`
	Verbose int    `short:"v" type:"counter" help:"Enable verbose logging."`
}

type CLI struct {
	GlobalArgs

	Api                api.ApiCmd                 `cmd:"" help:"Commands for working with the Foundry API."`
	Dump               cmds.DumpCmd               `cmd:"" help:"Dumps a project's blueprint to JSON."`
	CI                 cmds.CICmd                 `cmd:"" help:"Simulate a CI run."`
	ConfigureSatellite cmds.ConfigureSatelliteCmd `cmd:"" help:"Configure the local system to use a remote Earthly Satellite."`
	Mod                module.ModuleCmd           `kong:"cmd" help:"Commands for working with deployment modules."`
	Release            cmds.ReleaseCmd            `cmd:"" help:"Release a project."`
	Run                cmds.RunCmd                `cmd:"" help:"Run an Earthly target."`
	Scan               scan.ScanCmd               `cmd:"" help:"Commands for scanning for projects."`
	Secret             cmds.SecretCmd             `cmd:"" help:"Manage secrets."`
	Validate           cmds.ValidateCmd           `cmd:"" help:"Validates a project."`
	Version            VersionCmd                 `cmd:"" help:"Print the version."`

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

// AfterApply is called after CLI arguments are parsed.
// It is used to load the config and set up the run context.
func (c *CLI) AfterApply(kctx *kong.Context) error {
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
	store := secrets.NewDefaultSecretStore()
	cc := cuecontext.New()
	fs := billy.NewBaseOsFS()
	loader := project.NewDefaultProjectLoader(cc, store, logger, project.WithFs(fs))
	wlk := walker.NewCustomDefaultFSWalker(fs, logger)
	revWlk := walker.NewCustomReverseFSWalker(fs, logger)

	logger.Debug("attempting to load config")
	cfg, err := loadConfig(fs, logger)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.Debug("attempting to load root blueprint")
	rootProject, err := loadRootBlueprint(&loader, revWlk, logger)
	if err != nil {
		return fmt.Errorf("failed to load root blueprint: %w", err)
	}

	runctx := run.RunContext{
		ApiURL:                 cli.GlobalArgs.ApiURL,
		CI:                     cli.GlobalArgs.CI,
		Config:                 cfg,
		CueCtx:                 cc,
		FS:                     fs,
		FSWalker:               wlk,
		FSReverseWalker:        revWlk,
		Local:                  cli.GlobalArgs.Local,
		Logger:                 logger,
		ManifestGeneratorStore: deployment.NewDefaultManifestGeneratorStore(),
		ProjectLoader:          &loader,
		RootProject:            rootProject,
		SecretStore:            store,
		Verbose:                cli.GlobalArgs.Verbose,
	}

	kctx.Bind(runctx)
	return nil
}

var cli CLI

// Run is the entrypoint for the CLI tool.
func Run() int {
	cliArgs := os.Args[1:]

	parser := kong.Must(&cli,
		kong.Bind(run.RunContext{}),
		kong.Name("forge"),
		kong.Description("The CLI tool powering Catalyst Forge"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	kongplete.Complete(parser,
		kongplete.WithPredictor("path", complete.PredictFiles("*")),
	)

	ctx, err := parser.Parse(cliArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v\n", err)
		return 1
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "forge: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(Run())
}

func loadConfig(fs fs.Filesystem, logger *slog.Logger) (*config.CLIConfig, error) {
	cfg := config.NewCustomConfig(fs)
	exists, err := cfg.Exists()
	if err == nil && exists {
		logger.Debug("loading config")
		if err := cfg.Load(); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		logger.Debug("config not found")
	}

	return cfg, nil
}

func loadRootBlueprint(loader project.ProjectLoader, revWlk walker.FSReverseWalker, logger *slog.Logger) (*project.Project, error) {
	var rootProject *project.Project
	cwd, err := os.Getwd()
	if err != nil {
		logger.Warn("cannot load root blueprint: failed to get current working directory", "error", err)
	} else {
		repoRoot, err := git.FindGitRoot(cwd, &revWlk)
		if err != nil {
			logger.Warn("cannot load root blueprint: not in a git repository", "error", err)
		} else {
			p, err := loader.Load(repoRoot)
			if err != nil {
				logger.Warn("cannot load root blueprint: failed to load root blueprint", "error", err)
			}

			rootProject = &p
		}
	}

	return rootProject, nil
}
