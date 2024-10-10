package cmds

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	p "github.com/input-output-hk/catalyst-forge/lib/project/project"
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

	tagger := p.NewTagger(&project, ctx.CI, c.Trim, ctx.Logger)
	tagInfo, err := tagger.GetTagInfo()
	if err != nil {
		return fmt.Errorf("failed to get tag info: %w", err)
	}

	output.Generated = string(tagInfo.Generated)
	if tagInfo.Git.IsMono() {
		matches, err := project.MatchesTag(tagInfo.Git.ToMono())
		if err != nil {
			return fmt.Errorf("failed to match project tag: %w", err)
		} else if matches {
			if c.Trim {
				output.Git = tagInfo.Git.ToMono().Tag
			} else {
				output.Git = string(tagInfo.Git)
			}
		}
	} else {
		output.Git = string(tagInfo.Git)
	}

	printJson(output, c.Pretty)
	return nil
}
