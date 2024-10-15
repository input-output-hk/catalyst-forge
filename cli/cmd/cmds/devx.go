package cmds

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/command"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	// "github.com/willabides/kongplete"
	// "github.com/alecthomas/kong"
	// "github.com/posener/complete"
	// "github.com/willabides/kongplete"
)

type DevX struct {
	MarkdownPath string `arg:"" help:"Path to the markdown file." kong:"arg,predictor=file"`
	CommandName  string `arg:"" help:"Command to be executed." kong:"arg,predictor=devx-commands"`
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
