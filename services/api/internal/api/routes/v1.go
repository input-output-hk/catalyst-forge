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
	h := initializeHandlers(services, deps.Logger)

	// Register routes
	// Releases (minimal set; injection routes removed)
	v1 := r.Group("/api/v1")
	{
		releases := v1.Group("/releases")
		{
			releases.POST("", h.Release.Create)
			releases.GET(":release_id", h.Release.GetByID)
			releases.GET("", h.Release.List)
			releases.PATCH(":release_id", h.Release.Update)
			releases.DELETE(":release_id", h.Release.Delete)
			releases.GET(":release_id/modules", h.Release.GetModules)
			releases.POST(":release_id/modules", h.Release.AddModules)
			releases.DELETE(":release_id/modules/:module_key", h.Release.RemoveModule)
			releases.GET(":release_id/artifacts", h.Release.GetArtifacts)
			releases.POST(":release_id/artifacts", h.Release.AttachArtifact)
			releases.DELETE(":release_id/artifacts/:artifact_id", h.Release.DetachArtifact)
		}
	}

	RegisterDeployments(r, DeploymentDeps{H: h.Deployment})
	RegisterArtifacts(r, ArtifactDeps{H: h.Artifact})
	RegisterEnvironments(r, EnvironmentDeps{H: h.Environment})
	RegisterProjects(r, ProjectDeps{H: h.Project})
	RegisterRepositories(r, RepositoryDeps{H: h.Repository})
	RegisterTraces(r, TraceDeps{H: h.Trace})
	RegisterBuilds(r, BuildDeps{H: h.Build})
	RegisterRenderedReleases(r, RenderedReleaseDeps{H: h.Rendered})
	RegisterPromotions(r, PromotionDeps{H: h.Promotion})

	// Admin: organizations (removed)
}

// repositoryInstances holds all repository instances
type repositoryInstances struct {
	Artifact        artifactRepo.Repository
	Build           buildRepo.Repository
	Deployment      deploymentRepo.Repository
	Environment     environmentRepo.Repository
	GitOpsChange    gitopsRepo.Repository
	GitOpsSync      argoRepo.Repository
	Project         projectRepo.Repository
	Release         releaseRepo.Repository
	ReleaseModule   releaseRepo.ModuleRepository
	ReleaseArtifact releaseRepo.ArtifactRepository
	Repository      repositoryRepo.Repository
	Trace           traceRepo.Repository
	RenderedRelease releaseRepo.RenderedRepository
	Promotion       deploymentRepo.PromotionRepository
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
	Repository  repositoryService.Service
	Trace       traceService.Service
	Rendered    releaseService.RenderedService
	Promotion   deploymentService.PromotionService
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
	Rendered    *handlers.RenderedReleaseHandler
	Promotion   *handlers.PromotionHandler
}

// initializeRepositories creates all repository instances
func initializeRepositories(db *gorm.DB) repositoryInstances {
	return repositoryInstances{
		Artifact:        artifactRepo.NewRepository(db),
		Build:           buildRepo.NewRepository(db),
		Deployment:      deploymentRepo.NewRepository(db),
		Environment:     environmentRepo.NewRepository(db),
		GitOpsChange:    gitopsRepo.NewRepository(db),
		GitOpsSync:      argoRepo.NewRepository(db),
		Project:         projectRepo.NewRepository(db),
		Release:         releaseRepo.NewRepository(db),
		ReleaseModule:   releaseRepo.NewModuleRepository(db),
		ReleaseArtifact: releaseRepo.NewArtifactRepository(db),
		Repository:      repositoryRepo.NewRepository(db),
		Trace:           traceRepo.NewRepository(db),
		RenderedRelease: releaseRepo.NewRenderedRepository(db),
		Promotion:       deploymentRepo.NewPromotionRepository(db),
	}
}

// initializeServices creates all service instances
func initializeServices(txManager base.TxManager, repos repositoryInstances) serviceInstances {
	// Create deployment service
	deploymentSvc := deploymentService.NewService(
		txManager,
		repos.Deployment,
		repos.Release,
		repos.Environment,
	)

	// Create release service
	releaseSvc := releaseService.NewService(
		txManager,
		repos.Release,
		repos.ReleaseModule,
		repos.ReleaseArtifact,
		repos.Artifact,
	)

	// Create rendered release service
	renderedSvc := releaseService.NewRenderedService(txManager, repos.RenderedRelease)

	// Create promotion service
	promotionSvc := deploymentService.NewPromotionService(
		txManager,
		repos.Promotion,
		repos.Project,
		repos.Release,
		repos.Environment,
	)

	return serviceInstances{
		Artifact:    artifactService.NewService(txManager, repos.Artifact, repos.Build),
		Build:       buildService.NewService(txManager, repos.Build, repos.Project, repos.Repository),
		Deployment:  deploymentSvc,
		Environment: environmentService.NewService(txManager, repos.Environment),
		GitOps:      gitopsService.NewService(txManager, repos.GitOpsChange, repos.Deployment),
		Project:     projectService.NewService(repos.Project),
		Release:     releaseSvc,
		Repository:  repositoryService.NewService(repos.Repository),
		Trace:       traceService.NewService(repos.Trace),
		Rendered:    renderedSvc,
		Promotion:   promotionSvc,
	}
}

// initializeHandlers creates all handler instances
func initializeHandlers(services serviceInstances, logger *slog.Logger) handlerInstances {
	return handlerInstances{
		Release:     handlers.NewReleaseHandler(services.Release, logger),
		Deployment:  handlers.NewDeploymentHandler(services.Deployment, nil, logger),
		Artifact:    handlers.NewArtifactHandler(services.Artifact, logger),
		Environment: handlers.NewEnvironmentHandler(services.Environment, logger),
		Project:     handlers.NewProjectHandler(services.Project, logger),
		Repository:  handlers.NewRepositoryHandler(services.Repository, logger),
		Trace:       handlers.NewTraceHandler(services.Trace, logger),
		Build:       handlers.NewBuildHandler(services.Build, logger),
		Rendered:    handlers.NewRenderedReleaseHandler(services.Rendered, logger),
		Promotion:   handlers.NewPromotionHandler(services.Promotion, logger),
	}
}
