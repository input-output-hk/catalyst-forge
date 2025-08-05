package github

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/github"
)

type GetCmd struct {
	ID         *uint   `short:"i" help:"The ID of the authentication entry to retrieve."`
	Repository *string `short:"r" help:"The repository to retrieve the authentication entry for."`
	JSON       bool    `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *GetCmd) Run(ctx run.RunContext, cl client.Client) error {
	if c.ID == nil && c.Repository == nil {
		return fmt.Errorf("either --id or --repository must be specified")
	}

	if c.ID != nil && c.Repository != nil {
		return fmt.Errorf("only one of --id or --repository can be specified")
	}

	auth, err := c.retrieveAuth(cl)
	if err != nil {
		return err
	}

	if c.JSON {
		return outputJSON(auth)
	}

	return outputTable(auth)
}

func (c *GetCmd) retrieveAuth(cl client.Client) (*github.GithubRepositoryAuth, error) {
	if c.ID != nil {
		auth, err := cl.Github().GetAuth(context.Background(), *c.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get authentication entry by ID: %w", err)
		}
		return auth, nil
	}

	auth, err := cl.Github().GetAuthByRepository(context.Background(), *c.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication entry by repository: %w", err)
	}
	return auth, nil
}
