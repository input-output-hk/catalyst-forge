package release

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/github"
)

type ReleaseCreateCmd struct {
	Deploy  bool   `short:"d" help:"Automatically create a new deployment for the release."`
	Project string `arg:"" help:"The path to the project to create a new release for." kong:"arg,predictor=path"`
	Url     string `short:"u" help:"The URL to the Foundry API server (overrides global config)."`
}

func (c *ReleaseCreateCmd) Run(ctx run.RunContext) error {
	exists, err := fs.Exists(c.Project)
	if err != nil {
		return fmt.Errorf("could not check if project exists: %w", err)
	} else if !exists {
		return fmt.Errorf("project does not exist: %s", c.Project)
	}

	project, err := ctx.ProjectLoader.Load(c.Project)
	if err != nil {
		return fmt.Errorf("could not load project: %w", err)
	}

	commit, err := getCommitHash(project, ctx.Logger)
	if err != nil {
		return fmt.Errorf("could not get commit hash: %w", err)
	}

	branch, err := getBranch(project, ctx.Logger)
	if err != nil {
		return fmt.Errorf("could not get branch: %w", err)
	}

	// Only set the branch if it is not the default branch
	if branch == project.Blueprint.Global.Repo.DefaultBranch {
		branch = ""
	}

	path, err := project.GetRelativePath()
	if err != nil {
		return fmt.Errorf("could not get project path: %w", err)
	}

	var url string
	if c.Url == "" {
		if project.Blueprint.Global == nil ||
			project.Blueprint.Global.Deployment == nil ||
			project.Blueprint.Global.Deployment.Foundry.Api == "" {
			return errors.New("no foundry URL provided and no URL found in the root blueprint")
		}

		url = project.Blueprint.Global.Deployment.Foundry.Api
	} else {
		url = c.Url
	}

	client := api.NewClient(url, api.WithTimeout(10*time.Second))
	release, err := client.CreateRelease(context.Background(), &api.Release{
		SourceRepo:   project.Blueprint.Global.Repo.Url,
		SourceCommit: commit,
		SourceBranch: branch,
		Project:      project.Name,
		ProjectPath:  path,
		Bundle:       "something",
	}, c.Deploy)
	if err != nil {
		return fmt.Errorf("could not create release: %w", err)
	}

	if err := utils.PrintJson(release, true); err != nil {
		return err
	}

	return nil
}

func getCommitHash(project project.Project, logger *slog.Logger) (string, error) {
	if github.InGithubActions() {
		ghr := github.NewDefaultGithubRepo(logger)
		return ghr.GetCommit()
	}

	obj, err := project.Repo.HeadCommit()
	if err != nil {
		return "", err
	}

	return obj.Hash.String(), nil
}

func getBranch(project project.Project, logger *slog.Logger) (string, error) {
	if github.InGithubActions() {
		ghr := github.NewDefaultGithubRepo(logger)
		return ghr.GetBranch(), nil
	}

	return project.Repo.GetCurrentBranch()
}
