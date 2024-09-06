package cmds

import (
	"encoding/json"
	"fmt"
	"log/slog"

	blueprint "github.com/input-output-hk/catalyst-forge/blueprint/pkg/loader"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
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

// generateOpts generates the options for the Earthly executor based on the configuration file and flags.
func generateOpts(target string, flags *RunCmd, config *schema.Blueprint) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if config != nil {
		if _, ok := config.Project.CI.Targets[target]; ok {
			targetConfig := config.Project.CI.Targets[target]

			if len(targetConfig.Args) > 0 {
				var args []string
				for k, v := range targetConfig.Args {
					args = append(args, fmt.Sprintf("--%s", k), v)
				}

				opts = append(opts, earthly.WithTargetArgs(args...))
			}

			// We only run multiple platforms in CI mode to avoid issues with local builds.
			if targetConfig.Platforms != nil && flags.CI {
				opts = append(opts, earthly.WithPlatforms(targetConfig.Platforms...))
			}

			if targetConfig.Privileged != nil && *targetConfig.Privileged {
				opts = append(opts, earthly.WithPrivileged())
			}

			if targetConfig.Retries != nil {
				opts = append(opts, earthly.WithRetries(*targetConfig.Retries))
			}

			if len(targetConfig.Secrets) > 0 {
				opts = append(opts, earthly.WithSecrets(targetConfig.Secrets))
			}
		}

		if config.Global.CI.Providers.Earthly.Satellite != nil && !flags.Local {
			opts = append(opts, earthly.WithSatellite(*config.Global.CI.Providers.Earthly.Satellite))
		}
	}

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
	}

	return opts
}

// loadBlueprint loads the blueprint file from the given root path.
func loadBlueprint(rootPath string, logger *slog.Logger) (schema.Blueprint, error) {
	loader := blueprint.NewDefaultBlueprintLoader(rootPath, logger)

	err := loader.Load()
	if err != nil {
		return schema.Blueprint{}, fmt.Errorf("failed loading blueprint: %w", err)
	}

	config, err := loader.Decode()
	if err != nil {
		return schema.Blueprint{}, fmt.Errorf("failed decoding blueprint: %w", err)
	}

	return config, nil
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
