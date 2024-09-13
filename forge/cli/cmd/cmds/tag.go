package cmds

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/tag"
)

type TagCmd struct {
	CI      bool   `help:"Run in CI mode."`
	Pretty  bool   `short:"p" help:"Pretty print JSON output."`
	Project string `arg:"" help:"The project to generate tags for."`
	Trim    bool   `short:"t" help:"Trim the project path from the git tag."`
}

type TagOutput struct {
	Generated string `json:"generated"`
	Git       string `json:"git"`
}

func (c *TagCmd) Run(logger *slog.Logger) error {
	project, err := loadProject(c.Project, logger)
	if err != nil {
		return err
	}

	var output TagOutput
	tagger := tag.NewTagger(&project, c.CI, c.Trim, logger)

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
