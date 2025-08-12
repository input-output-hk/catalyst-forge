package module

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/generator"
	"github.com/spf13/cobra"
)

// TemplateOptions holds the flags for the module template command.
type TemplateOptions struct {
	Module  string
	Out     string
	SetPath map[string]string
}

// NewTemplateCommand creates the module template subcommand.
func NewTemplateCommand() *cobra.Command {
	opts := &TemplateOptions{
		SetPath: make(map[string]string),
	}

	cmd := &cobra.Command{
		Use:   "template PATH",
		Short: "Generates a project's (or module's) deployment YAML",
		Long:  `Generate Kubernetes deployment manifests from project modules with optional path overrides.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return templateExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Module, "module", "m", "", "The specific module to template")
	cmd.Flags().StringVarP(&opts.Out, "output", "o", "", "The output directory to write manifests to")
	cmd.Flags().StringToStringVar(&opts.SetPath, "set-path", nil, "Overrides the path for a given module (format: module=path)")

	return cmd
}

// templateExecute executes the module template command logic.
func templateExecute(ctx run.RunContext, path string, opts *TemplateOptions) error {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("could not stat path: %w", err)
	}

	var bundle deployment.ModuleBundle
	if stat.IsDir() {
		project, err := ctx.ProjectLoader.Load(path)
		if err != nil {
			return fmt.Errorf("could not load project: %w", err)
		}

		bundle = deployment.NewModuleBundle(&project)
	} else {
		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read file: %w", err)
		}

		bundle, err = deployment.ParseBundle(ctx.CueCtx, src)
		if err != nil {
			return fmt.Errorf("could not parse module file: %w", err)
		}
	}

	env, err := loadEnv(path, ctx.CueCtx, ctx.Logger)
	if err != nil {
		return fmt.Errorf("could not load environment file: %w", err)
	}

	manifests := make(map[string][]byte)
	gen := generator.NewGenerator(ctx.ManifestGeneratorStore, ctx.Logger)
	if opts.Module != "" {
		mod, ok := bundle.Bundle.Modules[opts.Module]
		if !ok {
			return fmt.Errorf("module %q not found", opts.Module)
		}

		if pathOverride, ok := opts.SetPath[opts.Module]; ok {
			ctx.Logger.Info("overriding path for module", "module", opts.Module, "path", pathOverride)
			mod.Path = pathOverride
		}

		raw := bundle.Raw.LookupPath(cue.ParsePath(fmt.Sprintf("modules.%s", opts.Module)))
		out, err := gen.Generate(mod, raw, bundle.Bundle.Env)
		if err != nil {
			return fmt.Errorf("failed to generate manifest: %w", err)
		}

		filename := fmt.Sprintf("%s.yaml", opts.Module)
		manifests[filename] = out
	} else {
		if opts.SetPath != nil {
			for name, pathOverride := range opts.SetPath {
				mod, ok := bundle.Bundle.Modules[name]
				if !ok {
					return fmt.Errorf("module %q not found", name)
				}

				mod.Path = pathOverride
				bundle.Bundle.Modules[name] = mod
			}
		}

		out, err := gen.GenerateBundle(bundle, env)
		if err != nil {
			return fmt.Errorf("failed to generate manifests: %w", err)
		}

		for name, manifest := range out.Manifests {
			filename := fmt.Sprintf("%s.yaml", name)
			manifests[filename] = manifest
		}
	}

	if opts.Out != "" {
		if err := writeManifests(opts.Out, manifests); err != nil {
			return fmt.Errorf("could not write manifests: %w", err)
		}
	} else {
		var output string
		for _, manifest := range manifests {
			output += fmt.Sprintf("%s\n---\n", strings.TrimSuffix(string(manifest), "---\n"))
		}

		fmt.Print(strings.TrimSuffix(output, "---\n"))
	}

	return nil
}

// loadEnv loads environment configuration from the deployment environment file.
func loadEnv(path string, ctx *cue.Context, logger *slog.Logger) (cue.Value, error) {
	var env cue.Value
	var envPath string

	filename := deployer.ENV_FILE
	stat, err := os.Stat(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("could not stat path: %w", err)
	}

	if stat.IsDir() {
		envPath = filepath.Join(path, filename)
	} else {
		envPath = filepath.Join(filepath.Dir(path), filename)
	}

	if _, err := os.Stat(envPath); err == nil {
		logger.Info("loading environment file", "path", envPath)
		contents, err := os.ReadFile(envPath)
		if err != nil {
			return cue.Value{}, fmt.Errorf("could not read environment file: %w", err)
		}

		env = ctx.CompileBytes(contents)
		if env.Err() != nil {
			return cue.Value{}, fmt.Errorf("could not compile environment file: %w", env.Err())
		}
	}

	return env, nil
}

// writeManifests writes deployment manifests to the specified directory.
func writeManifests(path string, manifests map[string][]byte) error {
	for name, manifest := range manifests {
		if err := os.WriteFile(filepath.Join(path, name), manifest, 0644); err != nil {
			return fmt.Errorf("could not write manifest: %w", err)
		}
	}

	return nil
}
