package project

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/google/go-github/v66/github"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	gh "github.com/input-output-hk/catalyst-forge/lib/tools/git/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/spf13/afero"
)

// RuntimeData is an interface for runtime data loaders.
type RuntimeData interface {
	Load(project *Project) map[string]cue.Value
}

// DeploymentRuntime is a runtime data loader for deployment related data.
type DeploymentRuntime struct {
	logger *slog.Logger
}

func (g *DeploymentRuntime) Load(project *Project) map[string]cue.Value {
	g.logger.Debug("Loading deployment runtime data")
	data := make(map[string]cue.Value)

	var registry string
	dc, err := project.RawBlueprint.Get("global.deployment.registries.containers").String()
	if err != nil {
		g.logger.Warn("Failed to get containers registry", "error", err)
	} else {
		registry = dc
	}

	var repo string
	rc, err := project.RawBlueprint.Get("global.repo.name").String()
	if err != nil {
		g.logger.Warn("Failed to get repository name", "error", err)
	} else {
		repo = rc
	}

	project.Blueprint = sb.Blueprint{
		Global: &sg.Global{
			Repo: &sg.Repo{
				Name: repo,
			},
		},
	}

	container := GenerateContainerName(project, project.Name, registry)
	data["CONTAINER_IMAGE"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, container))

	return data
}

// NewDeploymentRuntime creates a new DeploymentRuntime.
func NewDeploymentRuntime(logger *slog.Logger) *DeploymentRuntime {
	return &DeploymentRuntime{
		logger: logger,
	}
}

// GitRuntime is a runtime data loader for git related data.
type GitRuntime struct {
	fs     afero.Fs
	logger *slog.Logger
}

func (g *GitRuntime) Load(project *Project) map[string]cue.Value {
	g.logger.Debug("Loading git runtime data")
	data := make(map[string]cue.Value)

	hash, err := g.getCommitHash(project.Repo)
	if err != nil {
		g.logger.Warn("Failed to get commit hash", "error", err)
	} else {
		data["GIT_COMMIT_HASH"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, hash))
		data["GIT_HASH_OR_TAG"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, hash))
	}

	if project.Tag != nil {
		data["GIT_TAG"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, project.Tag.Full))
		data["GIT_TAG_VERSION"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, project.Tag.Version))
		data["GIT_HASH_OR_TAG"] = project.ctx.CompileString(fmt.Sprintf(`"%s"`, project.Tag.Version))
	} else {
		g.logger.Debug("No project tag found")
	}

	return data
}

// getCommitHash returns the commit hash of the HEAD commit.
func (g *GitRuntime) getCommitHash(repo *repo.GitRepo) (string, error) {
	env := gh.NewCustomGithubEnv(g.fs, g.logger)
	if env.HasEvent() {
		if env.GetEventType() == "pull_request" {
			g.logger.Debug("Found GitHub pull request event")
			event, err := env.GetEventPayload()
			if err != nil {
				return "", fmt.Errorf("failed to get event payload: %w", err)
			}

			pr, ok := event.(*github.PullRequestEvent)
			if !ok {
				return "", fmt.Errorf("unexpected event type")
			}

			if pr.PullRequest.Head.SHA == nil {
				return "", fmt.Errorf("pull request head SHA is empty")
			}

			return *pr.PullRequest.Head.SHA, nil
		} else if env.GetEventType() == "push" {
			g.logger.Debug("Found GitHub push event")
			event, err := env.GetEventPayload()
			if err != nil {
				return "", fmt.Errorf("failed to get event payload: %w", err)
			}

			push, ok := event.(*github.PushEvent)
			if !ok {
				return "", fmt.Errorf("unexpected event type")
			}

			if push.After == nil {
				return "", fmt.Errorf("push event after SHA is empty")
			}

			return *push.After, nil
		}
	}

	g.logger.Debug("No GitHub event found, getting commit hash from git repository")
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	obj, err := repo.GetCommit(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	return obj.Hash.String(), nil
}

// NewGitRuntime creates a new GitRuntime.
func NewGitRuntime(logger *slog.Logger) *GitRuntime {
	return &GitRuntime{
		fs:     afero.NewOsFs(),
		logger: logger,
	}
}

// NewCustomGitRuntime creates a new GitRuntime with a custom filesystem.
func NewCustomGitRuntime(fs afero.Fs, logger *slog.Logger) *GitRuntime {
	return &GitRuntime{
		fs:     fs,
		logger: logger,
	}
}
