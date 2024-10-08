package project

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	strats "github.com/input-output-hk/catalyst-forge/lib/project/project/strategies"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

// MonoTag represents a monorepo tag.
type MonoTag struct {
	Project string
	Tag     string
}

// TagInfo represents tag information.
type TagInfo struct {
	Generated string `json:"generated"`
	Git       string `json:"git"`
}

// Tagger parses tag information from projects.
type Tagger struct {
	ci      bool
	logger  *slog.Logger
	project *Project
	trim    bool
}

// GetTagInfo returns tag information for the project.
func (t *Tagger) GetTagInfo() (TagInfo, error) {
	gen, err := t.GenerateTag()
	if err != nil {
		return TagInfo{}, fmt.Errorf("failed to generate tag: %w", err)
	}

	git, err := t.GetGitTag()
	if err != nil {
		return TagInfo{}, fmt.Errorf("failed to get git tag: %w", err)
	}

	return TagInfo{
		Generated: gen,
		Git:       git,
	}, nil
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
	var gitTag string

	if t.ci {
		t.logger.Info("Parsing git tag from CI environment")
		gitTag = getGithubTag()
	} else {
		t.logger.Info("Parsing git tag from local environment")
		tag, err := getLocalTag(t.project.Repo)
		if err != nil {
			return "", err
		}

		gitTag = tag
	}

	if gitTag == "" {
		return "", nil
	} else {
		t.logger.Info("Found git tag", "tag", gitTag)
	}

	if IsMonoTag(gitTag) {
		mTag := ParseMonoTag(gitTag)
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

		if alias != "" {
			if relPath == alias {
				if t.trim {
					return mTag.Tag, nil
				} else {
					return gitTag, nil
				}
			} else {
				t.logger.Info("Alias does not match project path", "alias", alias, "project", relPath)
			}
		} else {
			if relPath == mTag.Project {
				if t.trim {
					return mTag.Tag, nil
				} else {
					return gitTag, nil
				}
			} else {
				t.logger.Info("Project path does not match monorepo tag", "path", relPath, "project", mTag.Project)
			}
		}
	} else {
		return gitTag, nil
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

// isMonoTag returns true if the tag is a monorepo tag.
func IsMonoTag(tag string) bool {
	parts := strings.Split(tag, "/")
	if len(parts) < 2 {
		return false
	} else {
		return true
	}
}

// parseMonoTag parses a monorepo tag into its project and tag components.
func ParseMonoTag(tag string) MonoTag {
	parts := strings.Split(tag, "/")
	return MonoTag{
		Project: strings.Join(parts[:len(parts)-1], "/"),
		Tag:     parts[len(parts)-1],
	}
}

// getLocalTag returns the tag of the current HEAD commit if it exists.
func getLocalTag(repo *gg.Repository) (string, error) {
	tags, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	var tag string
	err = tags.ForEach(func(t *plumbing.Reference) error {
		// Only process annotated tags
		tobj, err := repo.TagObject(t.Hash())
		if err != nil {
			return nil
		}

		if tobj.Target == ref.Hash() {
			tag = tobj.Name
			return nil
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to iterate over tags: %w", err)
	}

	return tag, nil
}

// GetGithubTag returns the tag from the GITHUB_REF environment variable if it
// exists.
func getGithubTag() string {
	tag, exists := os.LookupEnv("GITHUB_REF")
	if exists && strings.HasPrefix(tag, "refs/tags/") {
		return strings.TrimPrefix(tag, "refs/tags/")
	}

	return ""
}
