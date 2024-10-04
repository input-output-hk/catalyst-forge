package cmds

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type DevX struct {
	MarkdownPath string `arg:"" help:"Path to the markdown file."`
	CommandName  string `arg:"" help:"Command to be executed."`
}

func (c *DevX) Run(logger *slog.Logger, global GlobalArgs) error {
	// read the file from the specified path
	raw, err := os.ReadFile(c.MarkdownPath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %v", c.MarkdownPath, err)
	}

	// parse the file with prepared options
	commandGroups, err := extractCommandGroups(raw)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	// exec the command
	return processCmd(commandGroups, c.CommandName)
}

type command struct {
	content  string
	lang     *string
	platform *string
}

func (cmd *command) Exec() error {
	executor := getLangExecutor(cmd.lang)
	if executor == nil {
		return fmt.Errorf("only commands running with `sh` can be executed")
	}

	execCmd := exec.Command(cmd.content)

	output, err := execCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	fmt.Println(string(output))

	return nil
}

type commandGroup struct {
	name     string
	commands []command
}

func (cg *commandGroup) GetId() string {
	var result []rune

	for _, char := range cg.name {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			result = append(result, unicode.ToLower(char))
		} else if unicode.IsSpace(char) {
			result = append(result, '-')
		}
	}

	joined := string(result)

	re := regexp.MustCompile(`-+`)
	joined = re.ReplaceAllString(joined, "-")

	return strings.Trim(joined, "-")
}

func extractCommandGroups(data []byte) ([]commandGroup, error) {
	md := goldmark.New()
	reader := text.NewReader(data)
	doc := md.Parser().Parse(reader)

	// store the command groups and commands
	groups := []commandGroup{}
	var currentPlatform *string

	// walk through the ast nodes
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// look up for headers
		if heading, ok := n.(*ast.Heading); ok && entering {
			if heading.Level == 3 {
				currentPlatform = nil
				commandName := string(heading.Text(data))

				groups = append(groups, commandGroup{
					name:     commandName,
					commands: []command{},
				})
			}

			/* if heading.Level == 4 && len(groups) > 0 {
				platform := string(heading.Text(data))
				currentPlatform = &platform
			} */
		}

		// look up for code blocks
		if block, ok := n.(*ast.FencedCodeBlock); ok && entering && len(groups) > 0 {
			i := len(groups) - 1
			lang := string(block.Language(data))

			var buf bytes.Buffer
			for i := 0; i < block.Lines().Len(); i++ {
				line := block.Lines().At(i)
				buf.Write(line.Value(data))
			}

			groups[i].commands = append(groups[i].commands, command{
				content:  buf.String(),
				lang:     &lang,
				platform: currentPlatform,
			})
		}

		return ast.WalkContinue, nil
	})

	if len(groups) == 0 {
		return nil, errors.New("no command groups found in the markdown")
	}

	return groups, nil
}

func processCmd(list []commandGroup, cmd string) error {
	var foundCmd *command
	for _, v := range list {
		if v.GetId() == cmd {
			// TODO: should get the command corresponding to the current host platform
			foundCmd = &v.commands[0]
		}
	}

	if foundCmd == nil {
		return fmt.Errorf("command not found")
	}

	return foundCmd.Exec()
}

func getLangExecutor(lang *string) *string {
	if lang == nil {
		return nil
	}

	if *lang == "sh" {
		executor := "sh"
		return &executor
	} else {
		return nil
	}
}
