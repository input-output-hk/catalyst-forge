package utils

import (
	"errors"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
)

// GetFoundryURL retrieves the foundry URL from the given url or the root blueprint.
func GetFoundryURL(ctx run.RunContext, url string) (string, error) {
	if url == "" {
		bp, err := LoadRootBlueprint(ctx)
		if err != nil {
			return "", errors.New("no foundry URL provided and no root blueprint found")
		}

		if bp.Global == nil || bp.Global.Deployment == nil || bp.Global.Deployment.Foundry.Api == "" {
			return "", errors.New("no foundry URL provided in the root blueprint")
		}

		return bp.Global.Deployment.Foundry.Api, nil
	}

	return url, nil
}
