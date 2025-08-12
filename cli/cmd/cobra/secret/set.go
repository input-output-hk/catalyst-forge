package secret

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/spf13/cobra"
)

// SetOptions holds the flags for the secret set command.
type SetOptions struct {
	Field    []string
	Provider string
	Project  string
	Value    string
}

// NewSetCommand creates the secret set subcommand.
func NewSetCommand() *cobra.Command {
	opts := &SetOptions{
		Provider: "aws",
	}

	cmd := &cobra.Command{
		Use:   "set PATH [VALUE]",
		Short: "Set a secret",
		Long:  `Set secrets in configured providers using either a simple value or structured fields.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())

			if len(args) > 1 {
				opts.Value = args[1]
			}

			return setExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().StringSliceVarP(&opts.Field, "field", "f", nil, "A secret field to set (format: key=value)")
	cmd.Flags().StringVarP(&opts.Provider, "provider", "p", "aws", "The provider of the secret store")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Path to a project to use for getting secret configuration")

	return cmd
}

// setExecute executes the secret set command logic.
func setExecute(ctx run.RunContext, path string, opts *SetOptions) error {
	var secretPath, provider string

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
	} else {
		secretPath = path
		provider = opts.Provider
	}

	client, err := ctx.SecretStore.NewClient(ctx.Logger, secrets.Provider(provider))
	if err != nil {
		ctx.Logger.Error("Unable to create secret client.", "err", err)
		return fmt.Errorf("unable to create secret client: %w", err)
	}

	var data []byte
	if len(opts.Field) > 0 {
		fields := make(map[string]string)
		for _, f := range opts.Field {
			kv := strings.Split(f, "=")
			if len(kv) != 2 {
				return fmt.Errorf("invalid field format: %s: must be in the format of key=value", f)
			}

			fields[kv[0]] = kv[1]
		}

		data, err = json.Marshal(&fields)
		if err != nil {
			return err
		}
	} else {
		data = []byte(opts.Value)
	}

	id, err := client.Set(secretPath, string(data))
	if err != nil {
		ctx.Logger.Error("could not set secret", "err", err)
		return err
	}

	ctx.Logger.Info("Successfully set secret in AWS Secretsmanager.", "id", id)

	return nil
}
