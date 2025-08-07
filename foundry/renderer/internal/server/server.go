package server

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	
	"github.com/input-output-hk/catalyst-forge/foundry/renderer/internal/service"
	"github.com/input-output-hk/catalyst-forge/foundry/renderer/pkg/proto"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/external/kcl"
)

// Config holds the server configuration
type Config struct {
	Port      int
	Logger    *slog.Logger
	CachePath string
}

// Server represents the gRPC server
type Server struct {
	config  Config
	server  *grpc.Server
	logger  *slog.Logger
}

// NewServer creates a new gRPC server instance
func NewServer(config Config) (*Server, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	// Initialize cache directory if specified
	if config.CachePath != "" {
		if err := initializeCacheDirectory(config.CachePath, config.Logger); err != nil {
			return nil, fmt.Errorf("failed to initialize cache directory: %w", err)
		}
	}

	// Create the default manifest generator store with KCL caching options
	var storeOpts []deployment.Option
	if config.CachePath != "" {
		config.Logger.Info("Enabling KCL OCI module caching", "cachePath", config.CachePath)
		storeOpts = append(storeOpts, deployment.WithKCLOpts(kcl.WithCachePath(config.CachePath)))
	}

	store, err := deployment.NewDefaultManifestGeneratorStore(storeOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest generator store: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create and register the renderer service
	rendererService := service.NewRendererService(store, config.Logger)
	proto.RegisterRendererServiceServer(grpcServer, rendererService)

	// TODO: Add health check service if needed
	// grpc_health_v1.RegisterHealthServer(grpcServer, &healthService{})

	return &Server{
		config: config,
		server: grpcServer,
		logger: config.Logger.With("component", "grpc-server"),
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	address := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	s.logger.Info("Starting gRPC server", "address", address)
	
	if err := s.server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")
	s.server.GracefulStop()
}

// ForceStop forcefully stops the gRPC server
func (s *Server) ForceStop() {
	s.logger.Info("Force stopping gRPC server")
	s.server.Stop()
}

// healthService implements the gRPC health checking protocol
type healthService struct {
	grpc_health_v1.UnimplementedHealthServer
}

// initializeCacheDirectory creates the cache directory if it doesn't exist and validates permissions
func initializeCacheDirectory(cachePath string, logger *slog.Logger) error {
	// Check if directory already exists
	if info, err := os.Stat(cachePath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("cache path %s exists but is not a directory", cachePath)
		}
		logger.Debug("Cache directory already exists", "path", cachePath)
		
		// Test write permissions by creating a temporary file
		if err := testWritePermissions(cachePath); err != nil {
			return fmt.Errorf("cache directory %s is not writable: %w", cachePath, err)
		}
		
		return nil
	}

	// Create the directory with appropriate permissions
	logger.Info("Creating cache directory", "path", cachePath)
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cachePath, err)
	}

	// Verify the directory was created successfully
	if info, err := os.Stat(cachePath); err != nil {
		return fmt.Errorf("failed to verify cache directory creation: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("cache path %s is not a directory after creation", cachePath)
	}

	logger.Debug("Successfully created cache directory", "path", cachePath)
	return nil
}

// testWritePermissions tests if we can write to the cache directory
func testWritePermissions(cachePath string) error {
	testFile := filepath.Join(cachePath, ".renderer-test")
	
	// Try to create a test file
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot create test file: %w", err)
	}
	f.Close()
	
	// Clean up the test file
	if err := os.Remove(testFile); err != nil {
		// Log the error but don't fail - the main functionality works
		return fmt.Errorf("created test file but cannot remove it: %w", err)
	}
	
	return nil
}