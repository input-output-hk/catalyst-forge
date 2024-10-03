package cmds

import (
	"fmt"
	"log/slog"
	"os"
)

type DevX struct {
	MarkdownPath string `arg:"" help:"Path to the markdown file."`
	CommandName  string `arg:"" help:"Command to be executed."`
}

func (c *DevX) Run(logger *slog.Logger, global GlobalArgs) error {
	fmt.Println("MarkdownPath:", c.MarkdownPath, "CommandName:", c.CommandName)

	// read the file from the specified path
	raw, err := os.ReadFile(c.MarkdownPath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %v", c.MarkdownPath, err)
	}

	content := string(raw)

	fmt.Println(content)

	return nil
}
