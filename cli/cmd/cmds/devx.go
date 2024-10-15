package cmds

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/command"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type DevX struct {
	MarkdownPath string `kong:"arg,predictor=path" help:"Path to the markdown file."`
	CommandName  string `kong:"arg,predictor=devx-commands" help:"Command to be executed."`
}

func (c *DevX) Run(ctx run.RunContext, logger *slog.Logger) error {
	raw, err := os.ReadFile(c.MarkdownPath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %v", c.MarkdownPath, err)
	}

	prog, err := command.ExtractDevXMarkdown(raw)
	if err != nil {
		return err
	}

	return prog.ProcessCmd(c.CommandName, logger)
}
