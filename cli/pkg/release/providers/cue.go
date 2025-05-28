package providers

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/schema"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	lc "github.com/input-output-hk/catalyst-forge/lib/tools/cue"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

const CUE_BINARY = "cue"

type CueReleaserConfig struct {
	Version string `json:"version"`
}

type CueReleaser struct {
	config      CueReleaserConfig
	cue         executor.WrappedExecuter
	ecr         aws.ECRClient
	force       bool
	fs          fs.Filesystem
	handler     events.EventHandler
	logger      *slog.Logger
	project     project.Project
	release     sp.Release
	releaseName string
}

func (r *CueReleaser) Release() error {
	if !r.handler.Firing(&r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	registry := r.project.Blueprint.Global.Ci.Providers.Cue.Registry
	if registry == "" {
		return fmt.Errorf("must specify at least one CUE registry")
	}

	if r.config.Version == "" {
		return fmt.Errorf("no version specified")
	}

	var fullRegistry string
	prefix := r.project.Blueprint.Global.Ci.Providers.Cue.RegistryPrefix
	if prefix != "" {
		fullRegistry = fmt.Sprintf("%s/%s", registry, prefix)
	} else {
		fullRegistry = registry
	}

	module, err := r.loadModule()
	if err != nil {
		return fmt.Errorf("failed to load module: %w", err)
	}

	fullRepoName := fmt.Sprintf("%s/%s", registry, module)
	if aws.IsECRRegistry(fullRepoName) && schema.HasAWSProviderDefined(r.project.Blueprint) {
		r.logger.Info("Detected ECR registry, checking if repository exists", "registry", fullRepoName)
		if err := createECRRepoIfNotExists(r.ecr, &r.project, fullRepoName, r.logger); err != nil {
			return fmt.Errorf("failed to create ECR repository: %w", err)
		}
	}

	os.Setenv("CUE_REGISTRY", fullRegistry)
	defer os.Unsetenv("CUE_REGISTRY")

	path, err := r.project.GetRelativePath()
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	r.logger.Info("Publishing module", "path", path, "registry", fullRegistry, "version", r.config.Version)
	out, err := r.cue.Execute("mod", "publish", r.config.Version)
	if err != nil {
		r.logger.Error("Failed to publish module", "error", err, "output", string(out))
		return fmt.Errorf("failed to publish module: %w", err)
	}

	return nil
}

// loadModule loads the CUE module file.
func (r *CueReleaser) loadModule() (string, error) {
	modulePath := filepath.Join(r.project.Path, "cue.mod", "module.cue")
	if exists, err := r.fs.Exists(modulePath); err != nil {
		return "", fmt.Errorf("failed to check if module file exists: %w", err)
	} else if !exists {
		return "", fmt.Errorf("module file does not exist: %s", modulePath)
	}

	r.logger.Info("Loading module", "path", modulePath)
	contents, err := r.fs.ReadFile(modulePath)
	if err != nil {
		return "", fmt.Errorf("failed to read module file: %w", err)
	}

	v, err := lc.Compile(cuecontext.New(), contents)
	if err != nil {
		return "", fmt.Errorf("failed to parse module: %w", err)
	}

	moduleName := v.LookupPath(cue.ParsePath("module"))
	if !moduleName.Exists() {
		return "", fmt.Errorf("module file does not contain a module definition")
	}

	moduleNameString, err := moduleName.String()
	if err != nil {
		return "", fmt.Errorf("failed to get module name: %w", err)
	}

	return strings.Split(moduleNameString, "@")[0], nil
}

func NewCueReleaser(ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*CueReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	exec := executor.NewLocalExecutor(ctx.Logger, executor.WithWorkdir(project.Path))
	if _, ok := exec.LookPath(CUE_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find cue binary: %w", ok)
	}

	var config CueReleaserConfig
	if err := parseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	ecr, err := aws.NewECRClient(ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECR client: %w", err)
	}

	cue := executor.NewLocalWrappedExecutor(exec, CUE_BINARY)
	handler := events.NewDefaultEventHandler(ctx.Logger)
	return &CueReleaser{
		config:      config,
		cue:         cue,
		ecr:         ecr,
		fs:          billy.NewBaseOsFS(),
		force:       force,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     project,
		release:     release,
		releaseName: name,
	}, nil
}
