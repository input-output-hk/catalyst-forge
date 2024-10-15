package cmds

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/tui/ci"
)

type CICmd struct {
	Artifact string   `short:"a" help:"Dump all produced artifacts to the given path."`
	Path     string   `arg:"" default:"" help:"The path to scan from."`
	Platform []string `short:"p" help:"Run the target with the given platform."`
}

func (c *CICmd) Run(ctx run.RunContext) error {
	flags := RunCmd{
		Artifact: c.Artifact,
		Platform: c.Platform,
	}
	opts := generateOpts(&flags, ctx)
	return ci.Run(c.Path, ctx, opts...)
}
