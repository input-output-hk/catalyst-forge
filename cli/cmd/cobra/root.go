package cobra

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"cuelang.org/go/cue/cuecontext"
	"github.com/charmbracelet/log"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cobra/api"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cobra/module"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cobra/scan"
	"github.com/input-output-hk/catalyst-forge/cli/cmd/cobra/secret"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/config"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	schema "github.com/input-output-hk/catalyst-forge/lib/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Version = "dev"

	apiURL  string
	ciMode  bool
	local   bool
	verbose int
)

// NewRootCommand creates the root cobra command.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "forge",
		Short:             "The CLI tool powering Catalyst Forge",
		Long:              `Catalyst Forge CLI is a comprehensive command-line interface that serves as the primary user interaction point for the Catalyst Forge platform.`,
		PersistentPreRunE: initializeRunContext,
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "URL of the Foundry API (overrides the global configuration)")
	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "Run in CI mode")
	rootCmd.PersistentFlags().BoolVarP(&local, "local", "l", false, "Forces all runs to happen locally (ignores any remote satellites)")
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Enable verbose logging")

	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("ci", rootCmd.PersistentFlags().Lookup("ci"))
	viper.BindPFlag("local", rootCmd.PersistentFlags().Lookup("local"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(NewRunCommand())
	rootCmd.AddCommand(NewValidateCommand())
	rootCmd.AddCommand(NewDumpCommand())
	rootCmd.AddCommand(NewReleaseCommand())
	rootCmd.AddCommand(secret.NewCommand())
	rootCmd.AddCommand(NewCICommand())
	rootCmd.AddCommand(NewConfigureSatelliteCommand())
	rootCmd.AddCommand(scan.NewCommand())
	rootCmd.AddCommand(api.NewCommand())
	rootCmd.AddCommand(module.NewCommand())
	rootCmd.AddCommand(newCompletionCommand())

	return rootCmd
}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	viper.AddConfigPath("$HOME/.config/forge")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("FORGE")
	viper.AutomaticEnv()

	viper.ReadInConfig()
}

// initializeRunContext sets up the run context before command execution.
func initializeRunContext(cmd *cobra.Command, args []string) error {
	handler := log.New(os.Stderr)
	switch verbose {
	case 0:
		handler.SetLevel(log.FatalLevel)
	case 1:
		handler.SetLevel(log.WarnLevel)
	case 2:
		handler.SetLevel(log.InfoLevel)
	default:
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

	manifestStore, err := deployment.NewDefaultManifestGeneratorStore(deployment.WithKCLOpts())
	if err != nil {
		return fmt.Errorf("failed to create manifest store: %w", err)
	}

	runContext := run.RunContext{
		ApiURL:                 apiURL,
		CI:                     ciMode,
		Config:                 cfg,
		CueCtx:                 cc,
		FS:                     fs,
		FSWalker:               wlk,
		FSReverseWalker:        revWlk,
		Local:                  local,
		Logger:                 logger,
		ManifestGeneratorStore: manifestStore,
		ProjectLoader:          &loader,
		RootProject:            rootProject,
		SecretStore:            store,
		Verbose:                verbose,
	}

	cmd.SetContext(run.WithContext(cmd.Context(), runContext))

	return nil
}

// newVersionCommand creates the version command.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cuecontext.New()
			schema, err := schema.LoadSchema(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("forge version %s %s/%s\n", Version, runtime.GOOS, runtime.GOARCH)
			fmt.Printf("config schema version %s\n", schema.Version)
			return nil
		},
	}
}

// newCompletionCommand creates the shell completion command.
func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:
  $ source <(forge completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ forge completion bash > /etc/bash_completion.d/forge
  # macOS:
  $ forge completion bash > /usr/local/etc/bash_completion.d/forge

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ forge completion zsh > "${fpath[1]}/_forge"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ forge completion fish | source
  # To load completions for each session, execute once:
  $ forge completion fish > ~/.config/fish/completions/forge.fish

PowerShell:
  PS> forge completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> forge completion powershell > forge.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}

// loadConfig loads the CLI configuration from the filesystem.
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

// loadRootBlueprint loads the root project blueprint from the current git repository.
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
