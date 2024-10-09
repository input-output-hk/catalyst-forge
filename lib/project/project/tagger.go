package project

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	strats "github.com/input-output-hk/catalyst-forge/lib/project/project/tag"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
)

// Tagger parses tag information from projects.
type Tagger struct {
	ci      bool
	logger  *slog.Logger
	project *Project
	trim    bool
}

// GenerateTag generates a tag for the project based on the tagging strategy.
func (t *Tagger) GenerateTag() (string, error) {
	strategy := t.project.Blueprint.Global.CI.Tagging.Strategy

	t.logger.Info("Generating tag", "strategy", strategy)
	switch strategy {
	case schema.TagStrategyGitCommit:
		tag, err := strats.GitCommit(t.project.Repo)
		if err != nil {
			return "", err
		}

		return tag, nil
	default:
		return "", fmt.Errorf("unknown tag strategy: %s", strategy)
	}
}

// GetGitTag returns the git tag of the project.
// If the project is a monorepo, the tag is parsed and the project path is
// trimmed if necessary.
// If the project is not a monorepo, the tag is returned as is.
// If no git tag exists, or the project path does not match the monorepo tag,
// an empty string is returned.
func (t *Tagger) GetGitTag() (string, error) {

	gitTag, err := git.GetTag(t.project.Repo)
	if errors.Is(err, git.ErrTagNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get git tag: %w", err)
	}

	t.logger.Info("Found git tag", "tag", string(gitTag))

	if gitTag.IsMono() {
		mTag := gitTag.ToMono()
		t.logger.Info("Parsed monorepo tag", "project", mTag.Project, "tag", mTag.Tag)

		var alias string
		if t.project.Blueprint.Global.CI.Tagging.Aliases != nil {
			if _, ok := t.project.Blueprint.Global.CI.Tagging.Aliases[mTag.Project]; ok {
				t.logger.Info("Found alias", "project", mTag.Project, "alias", t.project.Blueprint.Global.CI.Tagging.Aliases[mTag.Project])
				alias = strings.TrimSuffix(t.project.Blueprint.Global.CI.Tagging.Aliases[mTag.Project], "/")
			}
		}

		relPath, err := t.project.GetRelativePath()
		if err != nil {
			return "", fmt.Errorf("failed to get project relative path: %w", err)
		}

		switch {
		case relPath == alias && t.trim:
			return mTag.Tag, nil
		case relPath == alias:
			return mTag.Full, nil
		case relPath == mTag.Project && t.trim:
			return mTag.Tag, nil
		case relPath == mTag.Project:
			return mTag.Full, nil
		default:
			t.logger.Info("Project path does not match monorepo tag", "path", relPath, "project", mTag.Project, "alias", alias)
		}
	} else {
		return string(gitTag), nil
	}

	return "", nil
}

// NewTagger creates a new tagger for the given project.
func NewTagger(p *Project, ci bool, trim bool, logger *slog.Logger) *Tagger {
	return &Tagger{
		ci:      ci,
		logger:  logger,
		project: p,
		trim:    trim,
	}
}
