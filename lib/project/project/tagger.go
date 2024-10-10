package project

import (
	"errors"
	"fmt"
	"log/slog"

	strats "github.com/input-output-hk/catalyst-forge/lib/project/project/strategies"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
)

// Tagger parses tag information from projects.
type Tagger struct {
	logger  *slog.Logger
	project *Project
}

// GetTagInfo returns tag information for the project.
func (t *Tagger) GetTagInfo() (TagInfo, error) {
	var gen, git git.Tag
	var err error

	if t.project.Blueprint.Global.CI.Tagging.Strategy != "" {
		gen, err = t.GenerateTag()
		if err != nil {
			return TagInfo{}, fmt.Errorf("failed to generate tag: %w", err)
		}
	} else {
		t.logger.Warn("No tag strategy defined, skipping tag generation")
	}

	git, err = t.GetGitTag()
	if err != nil {
		return TagInfo{}, fmt.Errorf("failed to get git tag: %w", err)
	}

	return TagInfo{
		Generated: gen,
		Git:       git,
	}, nil
}

// GenerateTag generates a tag for the project based on the tagging strategy.
func (t *Tagger) GenerateTag() (git.Tag, error) {
	strategy := t.project.Blueprint.Global.CI.Tagging.Strategy

	t.logger.Info("Generating tag", "strategy", strategy)
	switch strategy {
	case schema.TagStrategyGitCommit:
		tag, err := strats.GitCommit(t.project.Repo)
		if err != nil {
			return "", err
		}

		return git.Tag(tag), nil
	default:
		return "", fmt.Errorf("unknown tag strategy: %s", strategy)
	}
}

// GetGitTag returns the git tag of the project, if it exists.
func (t *Tagger) GetGitTag() (git.Tag, error) {

	gitTag, err := git.GetTag(t.project.Repo)
	if errors.Is(err, git.ErrTagNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get git tag: %w", err)
	}

	return gitTag, nil
}

// NewTagger creates a new tagger for the given project.
func NewTagger(p *Project, ci bool, trim bool, logger *slog.Logger) *Tagger {
	return &Tagger{
		logger:  logger,
		project: p,
	}
}
