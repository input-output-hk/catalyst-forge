package utils

import (
	"errors"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
)

// LoadRootBlueprint loads the root blueprint from the local git repository.
func LoadRootBlueprint(ctx run.RunContext) (blueprint.Blueprint, error) {
	gitRoot, err := git.FindGitRoot(".", ctx.ReverseWalker)
	if err != nil {
		return blueprint.Blueprint{}, errors.New("not in a git repository")
	}

	rbp, err := ctx.BlueprintLoader.Load(gitRoot, gitRoot)
	if err != nil {
		return blueprint.Blueprint{}, fmt.Errorf("failed to load root blueprint: %w", err)
	}

	if err := rbp.Validate(); err != nil {
		return blueprint.Blueprint{}, fmt.Errorf("failed to validate root blueprint: %w", err)
	}

	bp, err := rbp.Decode()
	if err != nil {
		return blueprint.Blueprint{}, fmt.Errorf("failed to decode root blueprint: %w", err)
	}

	return bp, nil
}
