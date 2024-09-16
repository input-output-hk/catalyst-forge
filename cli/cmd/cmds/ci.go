package cmds

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/tui/ci"
)

type CICmd struct {
	Artifact string   `short:"a" help:"Dump all produced artifacts to the given path."`
	Path     string   `arg:"" default:"" help:"The path to scan from."`
	Platform []string `short:"p" help:"Run the target with the given platform."`
}

func (c *CICmd) Run(logger *slog.Logger, global GlobalArgs) error {
	flags := RunCmd{
		Artifact: c.Artifact,
		Platform: c.Platform,
	}
	opts := generateOpts(&flags, &global)
	return ci.Run(c.Path, global.Local, opts...)
}
