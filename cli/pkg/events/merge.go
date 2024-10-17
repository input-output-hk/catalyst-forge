package events

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
)

// MergeEvent fires when a merge occurs to the default branch.
type MergeEvent struct {
	logger *slog.Logger
}

// MergeEventConfig is the configuration for the MergeEvent.
type MergeEventConfig struct {
	Branch string `json:"branch"`
}

func (m *MergeEvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	branch, err := git.GetBranch(p.Repo)
	if err != nil {
		return false, fmt.Errorf("failed to get branch: %w", err)
	}

	var targetBranch string
	if config.Exists() {
		var c MergeEventConfig
		if err := config.Decode(&c); err != nil {
			return false, fmt.Errorf("failed to decode event config: %w", err)
		}

		if c.Branch != "" {
			targetBranch = c.Branch
		} else {
			targetBranch = p.Blueprint.Global.Repo.DefaultBranch
		}
	} else {
		targetBranch = p.Blueprint.Global.Repo.DefaultBranch
	}

	m.logger.Debug("Checking branch", "branch", branch, "targetBranch", targetBranch)
	return branch == targetBranch, nil
}
