package cmds

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/project"
)

type GlobalArgs struct {
	CI      bool `help:"Run in CI mode."`
	Local   bool `short:"l" help:"Forces all runs to happen locally (ignores any remote satellites)."`
	Verbose int  `short:"v" type:"counter" help:"Enable verbose logging."`
}

// enumerate enumerates the Earthfile+Target pairs from the target map.
func enumerate(data map[string][]string) []string {
	var result []string
	for path, targets := range data {
		for _, target := range targets {
			result = append(result, fmt.Sprintf("%s+%s", path, target))
		}
	}

	return result
}

// generateOpts generates the options for the Earthly executor based on command
// flags.
func generateOpts(flags *RunCmd, global *GlobalArgs) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if flags != nil {
		if flags.Artifact != "" {
			opts = append(opts, earthly.WithArtifact(flags.Artifact))
		}

		if global.CI {
			opts = append(opts, earthly.WithCI())
		}

		// Users can explicitly set the platforms to use without being in CI mode.
		if flags.Platform != nil {
			opts = append(opts, earthly.WithPlatforms(flags.Platform...))
		}

		if len(flags.TargetArgs) > 0 && flags.TargetArgs[0] != "" {
			opts = append(opts, earthly.WithTargetArgs(flags.TargetArgs...))
		}
	}

	return opts
}

// loadProject loads the project from the given root path.
func loadProject(global GlobalArgs, rootPath string, logger *slog.Logger) (project.Project, error) {
	loader := project.NewDefaultProjectLoader(
		global.CI,
		global.Local,
		project.GetDefaultRuntimes(logger),
		logger,
	)
	return loader.Load(rootPath)
}

// printJson prints the given data as a JSON string.
func printJson(data interface{}, pretty bool) {
	var out []byte
	var err error

	if pretty {
		out, err = json.MarshalIndent(data, "", "  ")
	} else {
		out, err = json.Marshal(data)
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(out))
}
