package command

import (
	"bytes"
	"errors"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func ExtractDevXMarkdown(data []byte) (*Program, error) {
	md := goldmark.New()
	reader := text.NewReader(data)
	doc := md.Parser().Parse(reader)

	// store the command groups and commands
	groups := []CommandGroup{}
	var progName *string
	var currentPlatform *string

	// walk through the ast nodes
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// look up for headers
		if heading, ok := n.(*ast.Heading); ok && entering {
			if heading.Level == 1 {
				title := string(heading.Text(data))

				progName = &title
			}

			if heading.Level == 3 {
				currentPlatform = nil
				commandName := string(heading.Text(data))

				groups = append(groups, CommandGroup{
					name:     commandName,
					commands: []Command{},
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

			groups[i].commands = append(groups[i].commands, Command{
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
	if progName == nil {
		return nil, errors.New("no title found in the markdown")
	}

	prog := Program{
		name:   *progName,
		groups: groups,
	}

	return &prog, nil
}

func formatArgs(base []string, replacement string) []string {
	replaced := make([]string, len(base))

	for i, str := range base {
		replaced[i] = strings.ReplaceAll(str, "$", replacement)
	}

	return replaced
}
