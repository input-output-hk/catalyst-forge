package cmds

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/tag"
)

type TagCmd struct {
	Json    bool   `short:"j" help:"Output tags in JSON format."`
	Project string `arg:"" help:"The project to generate tags for."`
}

func (c *TagCmd) Run(logger *slog.Logger) error {
	project, err := loadProject(c.Project, logger)
	if err != nil {
		return err
	}

	var tags []string

	if project.Blueprint.Global.CI.Tagging.Strategy == "" {
		return fmt.Errorf("no tag strategy defined")
	}

	switch project.Blueprint.Global.CI.Tagging.Strategy {
	case schema.TagStrategyGitCommit:
		tag, err := tag.GitCommit(&project)
		if err != nil {
			return err
		}

		tags = append(tags, tag)
	default:
		return fmt.Errorf("unknown tag strategy: %s", project.Blueprint.Global.CI.Tagging.Strategy)
	}

	ref, err := project.Repo.Head()
	if err != nil {
		return err
	}

	t, err := tag.GetTag(project.Repo, ref)
	if err != nil {
		return err
	}

	fmt.Println("Found git tag: ", t)

	if c.Json {
		printJson(tags, false)
	} else {
		strTags := strings.Join(tags, " ")
		fmt.Println(strTags)
	}

	return nil
}
