package command

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func ExtractCommandMarkdown(data []byte) ([]commandGroup, error) {
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

func ProcessCmd(list []commandGroup, cmd string) error {
	var foundCmd *command
	for _, v := range list {
		if v.GetId() == cmd {
			// TODO: should get the fisrt (most specified) command corresponding to the current host platform
			foundCmd = &v.commands[0]
		}
	}

	if foundCmd == nil {
		return fmt.Errorf("command not found")
	}

	return foundCmd.Exec()
}

func GetLangExecutor(lang *string) (string, []string) {
	if lang == nil {
		return "", nil
	}

	// TODO: get more supported commands
	if *lang == "sh" {
		return "sh", []string{"-c", "$"}
	} else {
		return "", nil
	}
}

func FormatArgs(base []string, replacement string) []string {
	replaced := make([]string, len(base))

	for i, str := range base {
		replaced[i] = strings.ReplaceAll(str, "$", replacement)
	}

	return replaced
}
