package project

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/input-output-hk/catalyst-forge/lib/tools/argo"
	tg "github.com/input-output-hk/catalyst-forge/lib/tools/git"

	"cuelang.org/go/cue"
	"github.com/go-git/go-git/v5"
)

// RuntimeData is an interface for runtime data loaders.
type RuntimeData interface {
	Load(project *Project) map[string]cue.Value
}

// GitRuntime is a runtime data loader for git related data.
type GitRuntime struct {
	logger *slog.Logger
}

func (g *GitRuntime) Load(project *Project) map[string]cue.Value {
	g.logger.Debug("Loading git runtime data")
	data := make(map[string]cue.Value)

	hash, err := getCommitHash(project.Repo)
	if err != nil {
		g.logger.Warn("Failed to get commit hash", "error", err)
	} else {
		data["GIT_COMMIT_HASH"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, hash))
	}

	if project.Tag != nil {
		data["GIT_TAG"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, project.Tag.Full))
		data["GIT_TAG_VERSION"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, project.Tag.Version))
	} else {
		g.logger.Debug("No project tag found")
	}

	return data
}

// NewGitRuntime creates a new GitRuntime.
func NewGitRuntime(logger *slog.Logger) *GitRuntime {
	return &GitRuntime{
		logger: logger,
	}
}

// GetDefaultRuntimes returns the default runtime data loaders.
func GetDefaultRuntimes(logger *slog.Logger) []RuntimeData {
	return []RuntimeData{
		NewGitRuntime(logger),
	}
}

// getCommitHash returns the commit hash of the HEAD commit.
func getCommitHash(repo *git.Repository) (string, error) {
	if tg.InCI() {
		v, exists := os.LookupEnv("GITHUB_SHA")
		if !exists {
			return "", fmt.Errorf("GITHUB_SHA not found in environment")
		}

		return v, nil
	} else if argo.InArgo() {
		v, exists := os.LookupEnv("ARGOCD_APP_REVISION")
		if !exists {
			return "", fmt.Errorf("ARGOCD_APP_REVISION not found in environment")
		}

		return v, nil
	} else {
		ref, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}

		obj, err := repo.CommitObject(ref.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to get commit object: %w", err)
		}

		return obj.Hash.String(), nil
	}
}
