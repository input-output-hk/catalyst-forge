package cmds

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/forge/cli/tui/pipeline"
)

type CICmd struct {
	Artifact string   `short:"a" help:"Dump all produced artifacts to the given path."`
	Path     string   `arg:"" help:"The path to scan from."`
	Platform []string `short:"p" help:"Run the target with the given platform."`
}

func (c *CICmd) Run(logger *slog.Logger, global GlobalArgs) error {
	flags := RunCmd{
		Artifact: c.Artifact,
		Platform: c.Platform,
	}
	opts := generateOpts(&flags, &global)
	filters := []string{"^check.*$", "^build.*$", "^test.*$"}
	return pipeline.Start(c.Path, filters, global.Local, opts...)
}
