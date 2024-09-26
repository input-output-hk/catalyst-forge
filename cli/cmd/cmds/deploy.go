package cmds

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/deployment"
)

type DeployCmd struct {
	Project string `arg:"" help:"The path to the project to deploy."`
}

func (c *DeployCmd) Run(logger *slog.Logger, global GlobalArgs) error {
	project, err := loadProject(global, c.Project, logger)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	ctx := cuecontext.New()
	bundle, err := deployment.GenerateBundle(ctx, &project)
	if err != nil {
		return fmt.Errorf("could not generate bundle: %w", err)
	}

	src, err := bundle.Encode()
	if err != nil {
		return fmt.Errorf("could not encode bundle: %w", err)
	}
	fmt.Printf("bundle: %s\n", src)

	return nil
}
