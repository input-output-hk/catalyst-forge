package utils

import (
	"fmt"
	"reflect"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
)

// NewAPIClient creates a new API client.
func NewAPIClient(p *project.Project, ctx run.RunContext) (client.Client, error) {
	var apiURL string
	if ctx.ApiURL == "" {
		if !reflect.DeepEqual(p.Blueprint.Global, global.Global{}) &&
			!reflect.DeepEqual(p.Blueprint.Global.Ci, global.CI{}) &&
			!reflect.DeepEqual(p.Blueprint.Global.Ci.Providers, providers.Providers{}) &&
			!reflect.DeepEqual(p.Blueprint.Global.Ci.Providers.Foundry, providers.Foundry{}) {
			apiURL = p.Blueprint.Global.Ci.Providers.Foundry.Url
		} else {
			return nil, fmt.Errorf("no Foundry API URL found in the project")
		}
	} else {
		apiURL = ctx.ApiURL
	}

	var token string
	var opts []client.ClientOption
	exists, err := ctx.Config.Exists()
	if err != nil {
		return nil, fmt.Errorf("failed to check if config exists: %w", err)
	} else if exists {
		token = ctx.Config.Token
	}

	if token != "" {
		opts = append(opts, client.WithToken(token))
	} else {
		ctx.Logger.Warn("no token found in config, using anonymous access")
	}

	return client.NewClient(apiURL, opts...), nil
}
