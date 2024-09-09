package cmds

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
)

type TagCmd struct {
	Project string `arg:"" help:"The project to generate a tag for."`
}

func (c *TagCmd) Run(logger *slog.Logger) error {
	project, err := loadProject(c.Project, logger)
	if err != nil {
		return err
	}

	if project.Blueprint.Global.CI.Tagging.Strategy == "" {
		return fmt.Errorf("no tag strategy defined")
	}

	switch project.Blueprint.Global.CI.Tagging.Strategy {
	case schema.TagStrategyGitCommit:
		tag, err := strategyGitCommit(&project)
		if err != nil {
			return err
		}

		fmt.Println(tag)
	default:
		return fmt.Errorf("unknown tag strategy: %s", project.Blueprint.Global.CI.Tagging.Strategy)
	}

	return nil
}

func strategyGitCommit(project *project.Project) (string, error) {
	ref, err := project.Repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	obj, err := project.Repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	return obj.Hash.String(), nil
}
