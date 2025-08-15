package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	argoRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/argo"
	artifactRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/artifact"
	buildRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/build"
	deploymentRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/deployment"
	environmentRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/environment"
	gitopsRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/gitops"
	projectRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/project"
	releaseRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/release"
	repositoryRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/repository"
	traceRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/trace"
	artifactService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/artifact"
	buildService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/build"
	deploymentService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/deployment"
	environmentService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/environment"
	gitopsService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/gitops"
	projectService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/project"
	releaseService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/release"
	renderService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/render"
	repositoryService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/repository"
	traceService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/trace"
	"gorm.io/gorm"
)

// DomainDeps contains all dependencies for domain routes
type DomainDeps struct {
	DB     *gorm.DB
	Logger *slog.Logger
}

// RegisterDomainRoutes registers all domain routes
func RegisterDomainRoutes(r *gin.Engine, deps DomainDeps) {
	// Initialize transaction manager
	txManager := base.NewTxManager(deps.DB)

	// Initialize repositories
	repos := initializeRepositories(deps.DB)

	// Initialize services
	services := initializeServices(txManager, repos)

	// Initialize handlers
	handlers := initializeHandlers(services, deps.Logger)

	// Register routes
	RegisterReleases(r, ReleaseDeps{H: handlers.Release})
	RegisterDeployments(r, DeploymentDeps{H: handlers.Deployment})
	RegisterArtifacts(r, ArtifactDeps{H: handlers.Artifact})
	RegisterEnvironments(r, EnvironmentDeps{H: handlers.Environment})
	RegisterProjects(r, ProjectDeps{H: handlers.Project})
	RegisterRepositories(r, RepositoryDeps{H: handlers.Repository})
	RegisterTraces(r, TraceDeps{H: handlers.Trace})
	RegisterBuilds(r, BuildDeps{H: handlers.Build})
}

// repositoryInstances holds all repository instances
type repositoryInstances struct {
	Artifact         artifactRepo.Repository
	Build            buildRepo.Repository
	Deployment       deploymentRepo.Repository
	RenderJob        deploymentRepo.RenderJobRepository
	Environment      environmentRepo.Repository
	GitOpsChange     gitopsRepo.Repository
	ArgoSync         argoRepo.Repository
	Project          projectRepo.Repository
	Release          releaseRepo.Repository
	ReleaseModule    releaseRepo.ModuleRepository
	ReleaseInjection releaseRepo.InjectionRepository
	ReleaseArtifact  releaseRepo.ArtifactRepository
	Repository       repositoryRepo.Repository
	Trace            traceRepo.Repository
}

// serviceInstances holds all service instances
type serviceInstances struct {
	Artifact    artifactService.Service
	Build       buildService.Service
	Deployment  deploymentService.Service
	Environment environmentService.Service
	GitOps      gitopsService.Service
	Project     projectService.Service
	Release     releaseService.Service
	Render      renderService.Service
	Repository  repositoryService.Service
	Trace       traceService.Service
}

// handlerInstances holds all handler instances
type handlerInstances struct {
	Release     *handlers.ReleaseHandler
	Deployment  *handlers.DeploymentHandler
	Artifact    *handlers.ArtifactHandler
	Environment *handlers.EnvironmentHandler
	Project     *handlers.ProjectHandler
	Repository  *handlers.RepositoryHandler
	Trace       *handlers.TraceHandler
	Build       *handlers.BuildHandler
}

// initializeRepositories creates all repository instances
func initializeRepositories(db *gorm.DB) repositoryInstances {
	return repositoryInstances{
		Artifact:         artifactRepo.NewRepository(db),
		Build:            buildRepo.NewRepository(db),
		Deployment:       deploymentRepo.NewRepository(db),
		RenderJob:        deploymentRepo.NewRenderJobRepository(db),
		Environment:      environmentRepo.NewRepository(db),
		GitOpsChange:     gitopsRepo.NewRepository(db),
		ArgoSync:         argoRepo.NewRepository(db),
		Project:          projectRepo.NewRepository(db),
		Release:          releaseRepo.NewRepository(db),
		ReleaseModule:    releaseRepo.NewModuleRepository(db),
		ReleaseInjection: releaseRepo.NewInjectionRepository(db),
		ReleaseArtifact:  releaseRepo.NewArtifactRepository(db),
		Repository:       repositoryRepo.NewRepository(db),
		Trace:            traceRepo.NewRepository(db),
	}
}

// initializeServices creates all service instances
func initializeServices(txManager base.TxManager, repos repositoryInstances) serviceInstances {
	// Create render service first as it's needed by deployment service
	renderSvc := renderService.NewService(
		txManager,
		repos.RenderJob,
		repos.Deployment,
	)

	// Create deployment service
	deploymentSvc := deploymentService.NewService(
		txManager,
		repos.Deployment,
		repos.Release,
		repos.Environment,
		repos.RenderJob,
	)

	// Create release service
	releaseSvc := releaseService.NewService(
		txManager,
		repos.Release,
		repos.ReleaseModule,
		repos.ReleaseInjection,
		repos.ReleaseArtifact,
		repos.Artifact,
	)

	return serviceInstances{
		Artifact:    artifactService.NewService(txManager, repos.Artifact, repos.Build),
		Build:       buildService.NewService(txManager, repos.Build, repos.Project, repos.Repository),
		Deployment:  deploymentSvc,
		Environment: environmentService.NewService(txManager, repos.Environment),
		GitOps:      gitopsService.NewService(txManager, repos.GitOpsChange, repos.Deployment),
		Project:     projectService.NewService(repos.Project),
		Release:     releaseSvc,
		Render:      renderSvc,
		Repository:  repositoryService.NewService(repos.Repository),
		Trace:       traceService.NewService(repos.Trace),
	}
}

// initializeHandlers creates all handler instances
func initializeHandlers(services serviceInstances, logger *slog.Logger) handlerInstances {
	return handlerInstances{
		Release:     handlers.NewReleaseHandler(services.Release, logger),
		Deployment:  handlers.NewDeploymentHandler(services.Deployment, services.Render, logger),
		Artifact:    handlers.NewArtifactHandler(services.Artifact, logger),
		Environment: handlers.NewEnvironmentHandler(services.Environment, logger),
		Project:     handlers.NewProjectHandler(services.Project, logger),
		Repository:  handlers.NewRepositoryHandler(services.Repository, logger),
		Trace:       handlers.NewTraceHandler(services.Trace, logger),
		Build:       handlers.NewBuildHandler(services.Build, logger),
	}
}
