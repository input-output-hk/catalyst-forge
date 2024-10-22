package cmds

import (
	"fmt"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/command"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type DevX struct {
	Discover     bool   `kong:"short=d" help:"List all markdown files available in the project or all commands in the markdown file if file is specified."`
	MarkdownPath string `kong:"arg,predictor=path,optional" help:"Path to the markdown file."`
	CommandName  string `kong:"arg,optional" help:"Command to be executed."`
}

func (c *DevX) Run(ctx run.RunContext) error {
	// for the "-d" flag
	if c.Discover {
		return nil
	}

	// validate args if without "-d" flag
	if c.MarkdownPath == "" {
		return fmt.Errorf("expected \"<markdown-path> <command-name>\"")
	}
	if c.MarkdownPath != "" && c.CommandName == "" {
		return fmt.Errorf("expected \"<command-name>\"")
	}

	// read the markdown and execute the command
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
