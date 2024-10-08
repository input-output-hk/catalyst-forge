package cmds

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type DevX struct {
	MarkdownPath string `arg:"" help:"Path to the markdown file."`
	CommandName  string `arg:"" help:"Command to be executed."`
}

func (c *DevX) Run(ctx run.RunContext, logger *slog.Logger) error {
	// read the file from the specified path
	raw, err := os.ReadFile(c.MarkdownPath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %v", c.MarkdownPath, err)
	}

	// parse the file with prepared options
	commandGroups, err := extractCommandGroups(raw)
	if err != nil {
		return err
	}

	// exec the command
	return processCmd(commandGroups, c.CommandName)
}
