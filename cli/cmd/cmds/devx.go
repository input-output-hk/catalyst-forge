package cmds

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type DevX struct {
	MarkdownPath string `arg:"" help:"Path to the markdown file."`
	CommandName  string `arg:"" help:"Command to be executed."`
}

type command struct {
	content  string
	lang     *string
	platform *string
}

type commandGroup struct {
	name     string
	commands []command
}

func (c *DevX) Run(logger *slog.Logger, global GlobalArgs) error {
	fmt.Println("MarkdownPath:", c.MarkdownPath, "CommandName:", c.CommandName)

	// read the file from the specified path
	raw, err := os.ReadFile(c.MarkdownPath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %v", c.MarkdownPath, err)
	}

	// parse the file with prepared options
	commandGroups, err := extractCommandGroups(raw)
	if err != nil {
		return fmt.Errorf("could not extract commands: %v", err)
	}

	// Output the command groups
	for _, group := range commandGroups {
		fmt.Printf("Command Group: %s\n", group.name)
		fmt.Printf("Command Count: %v\n", len(group.commands))
		for _, cmd := range group.commands {
			fmt.Printf("  Command (lang: %s, platform: %s):\n%s\n", *cmd.lang, *cmd.platform, cmd.content)
		}
	}

	return nil
}

func extractCommandGroups(data []byte) ([]commandGroup, error) {
	md := goldmark.New()
	reader := text.NewReader(data)
	doc := md.Parser().Parse(reader)

	// store the command groups and commands
	var groups []commandGroup
	var currentGroup *commandGroup
	var currentPlatform *string

	// walk through the ast nodes
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if heading, ok := n.(*ast.Heading); ok && entering {
			if heading.Level == 2 {
				currentPlatform = nil
				commandName := string(heading.Text(data))

				currentGroup = &commandGroup{
					name:     commandName,
					commands: []command{},
				}
				groups = append(groups, *currentGroup)
			}

			if heading.Level == 3 && currentGroup != nil {
				platform := string(heading.Text(data))
				currentPlatform = &platform
			}
		}

		if block, ok := n.(*ast.FencedCodeBlock); ok && entering && currentGroup != nil {
			lang := string(block.Language(data))

			var buf bytes.Buffer
			for i := 0; i < block.Lines().Len(); i++ {
				line := block.Lines().At(i)
				buf.Write(line.Value(data))
			}

			currentGroup.commands = append(currentGroup.commands, command{
				content:  buf.String(),
				lang:     &lang,
				platform: currentPlatform,
			})

			fmt.Println(len(currentGroup.commands))
		}

		return ast.WalkContinue, nil
	})

	// Check if any groups were found
	if len(groups) == 0 {
		return nil, errors.New("no command groups found in the markdown")
	}

	return groups, nil
}
