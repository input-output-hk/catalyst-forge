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
	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return err
	}

	var output TagOutput
	tagger := p.NewTagger(&project, ctx.CI, c.Trim, ctx.Logger)

	if project.Blueprint.Global.CI.Tagging.Strategy != "" {
		tag, err := tagger.GenerateTag()
		if err != nil {
			return fmt.Errorf("failed to generate tag: %w", err)
		}

		output.Generated = tag
	}

	gitTag, err := tagger.GetGitTag()
	if err != nil {
		return fmt.Errorf("failed to get git tag: %w", err)
	}
	output.Git = gitTag

	printJson(output, c.Pretty)
	return nil
}
