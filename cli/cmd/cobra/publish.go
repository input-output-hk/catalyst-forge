package cobra

import (
    "fmt"

    "github.com/input-output-hk/catalyst-forge/cli/pkg/publish"
    "github.com/input-output-hk/catalyst-forge/cli/pkg/run"
    "github.com/input-output-hk/catalyst-forge/lib/tools/fs"
    "github.com/spf13/cobra"
)

// PublishOptions holds the flags for the publish command.
type PublishOptions struct {
    Force bool
}

// NewPublishCommand creates the publish command.
func NewPublishCommand() *cobra.Command {
    opts := &PublishOptions{}

    cmd := &cobra.Command{
        Use:   "publish PROJECT TARGET",
        Short: "Publish a project's artifacts",
        Long:  `Execute artifact publishing using configured providers with automatic CI mode activation.`,
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := run.MustFromContext(cmd.Context())
            return publishExecute(ctx, args[0], args[1], opts)
        },
    }

    cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force the publish to run")

    return cmd
}

// publishExecute executes the publish command logic.
func publishExecute(ctx run.RunContext, projectPath, targetName string, opts *PublishOptions) error {
    exists, err := fs.Exists(projectPath)
    if err != nil {
        return fmt.Errorf("could not check if project exists: %w", err)
    } else if !exists {
        return fmt.Errorf("project does not exist: %s", projectPath)
    }

    project, err := ctx.ProjectLoader.Load(projectPath)
    if err != nil {
        return err
    }

    // Validate the target exists in the blueprint under Project.Publishers
    publisherDef, ok := project.Blueprint.Project.Publishers[targetName]
    if !ok {
        return fmt.Errorf("unknown publish target: %s", targetName)
    }

    // Always publish in CI mode
    ctx.CI = true
    store := publish.NewDefaultPublisherStore()
    pubType := publish.PublisherType(publisherDef.Type)
    publisherRunner, err := store.GetPublisher(
        pubType,
        ctx,
        project,
        targetName,
        opts.Force,
    )
    if err != nil {
        return fmt.Errorf("failed to initialize publisher: %w", err)
    }

    if err := publisherRunner.Publish(); err != nil {
        return fmt.Errorf("failed to publish: %w", err)
    }

    return nil
}
