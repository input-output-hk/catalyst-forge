package events

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
)

// MergeEvent fires when a merge occurs to the default branch.
type MergeEvent struct {
	logger  *slog.Logger
	project *project.Project
}

func (m *MergeEvent) Firing() (bool, error) {
	branch, err := git.GetBranch(m.project.Repo)
	if err != nil {
		return false, fmt.Errorf("failed to get branch: %w", err)
	}

	m.logger.Debug("Checking branch", "branch", branch, "default", m.project.Blueprint.Global.Repo.DefaultBranch)
	return branch == m.project.Blueprint.Global.Repo.DefaultBranch, nil
}