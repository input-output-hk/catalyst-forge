package cmds

import (
	"fmt"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/command"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type DevX struct {
	MarkdownPath string `arg:"" help:"Path to the markdown file."`
	CommandName  string `arg:"" help:"Command to be executed."`
}

func (c *DevX) Run(ctx run.RunContext) error {
	raw, err := os.ReadFile(c.MarkdownPath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %v", c.MarkdownPath, err)
	}

	prog, err := command.ExtractDevXMarkdown(raw)
	if err != nil {
		return err
	}

	return prog.ProcessCmd(c.CommandName, ctx.Logger)
}
