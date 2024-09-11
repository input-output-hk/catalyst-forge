package cmds

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	ptag "github.com/input-output-hk/catalyst-forge/forge/cli/pkg/tag"
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

	if project.Blueprint.Global.CI.Tagging.Strategy == "" {
		return fmt.Errorf("no tag strategy defined")
	}

	switch project.Blueprint.Global.CI.Tagging.Strategy {
	case schema.TagStrategyGitCommit:
		tag, err := ptag.GitCommit(&project)
		if err != nil {
			return err
		}

		output.Generated = tag

		logger.Info("Generated tag", "tag", tag, "strategy", project.Blueprint.Global.CI.Tagging.Strategy)
	default:
		return fmt.Errorf("unknown tag strategy: %s", project.Blueprint.Global.CI.Tagging.Strategy)
	}

	gitTag, err := parseGitTag(project, c.CI)
	if err != nil {
		return fmt.Errorf("failed to parse git tag: %w", err)
	}

	if gitTag != "" {
		logger.Info("Found git tag", "tag", gitTag)
	} else {
		logger.Info("No git tag found")
	}

	if gitTag != "" && ptag.IsMonoTag(gitTag) {
		tag, err := ptag.ParseMonoTag(gitTag)
		if err != nil {
			return fmt.Errorf("failed to parse monorepo tag: %w", err)
		}

		logger.Info("Found monorepo tag", "project", tag.Project, "tag", tag.Tag)

		finalTag, err := handleMonoTag(project, tag, c.Trim, logger)
		if err != nil {
			return fmt.Errorf("failed to parse monorepo tag: %w", err)
		}

		if finalTag != "" {
			output.Git = finalTag
		} else {
			logger.Warn("Tag does not match project")
		}
	} else if gitTag != "" {
		logger.Warn("Git tag is not a monorepo tag", "tag", gitTag)
		output.Git = gitTag
	}

	printJson(output, c.Pretty)
	return nil
}

// parseGitTag returns the git tag if it exists.
func parseGitTag(project project.Project, ci bool) (string, error) {
	if ci {
		t, exists := os.LookupEnv("GITHUB_REF")
		if exists && strings.HasPrefix(t, "refs/tags/") {
			return strings.TrimPrefix(t, "refs/tags/"), nil
		}

		return "", nil
	} else {
		tag, err := ptag.GetTag(project.Repo)
		if err != nil {
			return "", err
		}

		return tag, nil
	}
}

// handleMonoTag returns the final tag if the project path matches the monorepo tag.
func handleMonoTag(project project.Project, tag ptag.MonoTag, trim bool, logger *slog.Logger) (string, error) {
	projectPath, err := filepath.Abs(project.Path)
	if err != nil {
		return "", fmt.Errorf("failed to get project path: %w", err)
	}

	repoRoot, err := filepath.Abs(project.RepoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get repo root: %w", err)
	}

	if !strings.HasPrefix(projectPath, repoRoot) {
		return "", fmt.Errorf("project path is not a subdirectory of the repo root")
	}

	relPath, err := filepath.Rel(repoRoot, projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Check if the project has an alias
	var preAlias string
	if project.Blueprint.Global.CI.Tagging.Aliases != nil {
		if _, ok := project.Blueprint.Global.CI.Tagging.Aliases[tag.Project]; ok {
			logger.Info("Found alias", "project", tag.Project, "alias", project.Blueprint.Global.CI.Tagging.Aliases[tag.Project])
			preAlias = tag.Project
			tag.Project = project.Blueprint.Global.CI.Tagging.Aliases[tag.Project]
		}
	}

	if !ptag.MatchMonoTag(relPath, tag) {
		return "", nil
	} else {
		if trim {
			return tag.Tag, nil
		} else {
			if preAlias != "" {
				return preAlias + "/" + tag.Tag, nil
			} else {
				return tag.Project + "/" + tag.Tag, nil
			}
		}
	}
}
