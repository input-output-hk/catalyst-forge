package secret

import (
	"encoding/json"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/spf13/cobra"
)

// GetOptions holds the flags for the secret get command.
type GetOptions struct {
	Key      string
	Project  string
	Provider string
}

// NewGetCommand creates the secret get subcommand.
func NewGetCommand() *cobra.Command {
	opts := &GetOptions{
		Provider: "aws",
	}

	cmd := &cobra.Command{
		Use:   "get PATH",
		Short: "Get a secret",
		Long:  `Retrieve secrets from configured providers, with optional project-based configuration.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return getExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Key, "key", "k", "", "The key inside of the secret to get")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Path to a project to use for getting secret configuration")
	cmd.Flags().StringVarP(&opts.Provider, "provider", "p", "aws", "The provider of the secret store")

	return cmd
}

// getExecute executes the secret get command logic.
func getExecute(ctx run.RunContext, path string, opts *GetOptions) error {
	var secretPath, provider string
	var maps map[string]string

	if opts.Project != "" {
		exists, err := fs.Exists(opts.Project)
		if err != nil {
			return fmt.Errorf("could not check if project exists: %w", err)
		} else if !exists {
			return fmt.Errorf("project does not exist: %s", opts.Project)
		}

		project, err := ctx.ProjectLoader.Load(opts.Project)
		if err != nil {
			return fmt.Errorf("could not load project: %w", err)
		}

		var secret sc.Secret
		if err := project.Raw().DecodePath(path, &secret); err != nil {
			return fmt.Errorf("could not decode secret: %w", err)
		}

		secretPath = secret.Path
		provider = secret.Provider

		if len(secret.Maps) > 0 {
			maps = secret.Maps
		} else {
			maps = make(map[string]string)
		}
	} else {
		secretPath = path
		provider = opts.Provider
		maps = make(map[string]string)
	}

	client, err := ctx.SecretStore.NewClient(ctx.Logger, secrets.Provider(provider))
	if err != nil {
		ctx.Logger.Error("Unable to create secret client.", "err", err)
		return fmt.Errorf("unable to create secret client: %w", err)
	}

	s, err := client.Get(secretPath)
	if err != nil {
		return fmt.Errorf("could not get secret: %w", err)
	}

	if len(maps) > 0 {
		mappedSecret := make(map[string]string)
		m := make(map[string]string)

		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return err
		}

		for k, v := range maps {
			if _, ok := m[v]; !ok {
				return fmt.Errorf("key %s not found in secret at %s", v, secretPath)
			}

			mappedSecret[k] = m[v]
		}

		if opts.Key != "" {
			if _, ok := mappedSecret[opts.Key]; !ok {
				return fmt.Errorf("key %s not found in mapped secret at %s", opts.Key, secretPath)
			}

			fmt.Println(mappedSecret[opts.Key])
			return nil
		} else {
			utils.PrintJson(mappedSecret, false)
			return nil
		}
	}

	if opts.Key != "" {
		m := make(map[string]string)

		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return err
		}

		if _, ok := m[opts.Key]; !ok {
			return fmt.Errorf("key %s not found in secret at %s", opts.Key, secretPath)
		}

		fmt.Println(m[opts.Key])
	} else {
		fmt.Println(s)
	}
	return nil
}
