package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

type TagCmd struct {
	Pretty  bool   `short:"p" help:"Pretty print JSON output."`
	Project string `arg:"" help:"The project to generate tags for."`
	Trim    bool   `short:"t" help:"Trim the project path from the git tag."`
}

type TagOutput struct {
	Generated string `json:"generated"`
	Git       string `json:"git"`
}

func (c *TagCmd) Run(ctx run.RunContext) error {
	var output TagOutput

	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return err
	}

	if project.TagInfo == nil {
		return fmt.Errorf("failed to get tag info")
	}

	output.Generated = string(project.TagInfo.Generated)
	matches, err := project.TagMatches()
	if err != nil {
		return fmt.Errorf("failed to check if tag matches: %w", err)
	}

	if matches {
		if project.TagInfo.Git.IsMono() {
			m := project.TagInfo.Git.ToMono()
			if c.Trim {
				output.Git = m.Tag
			} else {
				output.Git = m.Full
			}
		} else {
			output.Git = string(project.TagInfo.Git)
		}
	}

	printJson(output, c.Pretty)
	return nil
}
