package cmds

import (
	"encoding/json"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
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
func generateOpts(flags *RunCmd, ctx run.RunContext) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if flags != nil {
		if flags.Artifact != "" {
			opts = append(opts, earthly.WithArtifact(flags.Artifact))
		}

		if ctx.CI {
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
