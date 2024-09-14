package cmds

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
)

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
func generateOpts(flags *RunCmd) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if flags != nil {
		if flags.Artifact != "" {
			opts = append(opts, earthly.WithArtifact(flags.Artifact))
		}

		if flags.CI {
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
func loadProject(rootPath string, logger *slog.Logger) (project.Project, error) {
	loader := project.NewDefaultProjectLoader(loadRuntimes(logger), logger)
	return loader.Load(rootPath)
}

// loadRuntimes loads the all runtime data collectors.
func loadRuntimes(logger *slog.Logger) []project.RuntimeData {
	return []project.RuntimeData{
		project.NewGitRuntime(logger),
	}
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
