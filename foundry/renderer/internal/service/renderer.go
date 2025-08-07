package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/foundry/renderer/pkg/proto"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/generator"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/proto/generated/project"
)

// RendererService implements the gRPC RendererService
type RendererService struct {
	proto.UnimplementedRendererServiceServer
	generator generator.Generator
	logger    *slog.Logger
	ctx       *cue.Context
}

// NewRendererService creates a new RendererService instance
func NewRendererService(store deployment.ManifestGeneratorStore, logger *slog.Logger) *RendererService {
	if logger == nil {
		logger = slog.Default()
	}

	return &RendererService{
		generator: generator.NewGenerator(store, logger),
		logger:    logger.With("service", "renderer"),
		ctx:       cuecontext.New(),
	}
}

// RenderManifests implements the RenderManifests gRPC method
func (s *RendererService) RenderManifests(ctx context.Context, req *proto.RenderManifeststRequest) (*proto.RenderManifestsResponse, error) {
	s.logger.Info("Received render manifests request")

	if req.Bundle == nil {
		err := fmt.Errorf("bundle is required")
		s.logger.Error("Invalid request", "error", err)
		return &proto.RenderManifestsResponse{
			Error: err.Error(),
		}, nil
	}

	// Convert protobuf bundle to deployment.ModuleBundle
	bundle, err := s.convertProtoBundle(req.Bundle)
	if err != nil {
		s.logger.Error("Failed to convert proto bundle", "error", err)
		return &proto.RenderManifestsResponse{
			Error: fmt.Sprintf("Failed to convert bundle: %v", err),
		}, nil
	}

	// Parse environment data if provided
	var env cue.Value
	if len(req.EnvData) > 0 {
		env = s.ctx.CompileBytes(req.EnvData)
		if env.Err() != nil {
			err := fmt.Errorf("failed to parse environment data: %w", env.Err())
			s.logger.Error("Invalid environment data", "error", err)
			return &proto.RenderManifestsResponse{
				Error: err.Error(),
			}, nil
		}
	}

	// Generate manifests using the deployment generator
	result, err := s.generator.GenerateBundle(bundle, env)
	if err != nil {
		s.logger.Error("Failed to generate manifests", "error", err)
		return &proto.RenderManifestsResponse{
			Error: fmt.Sprintf("Failed to generate manifests: %v", err),
		}, nil
	}

	// Convert results to protobuf format
	manifests := make(map[string][]byte)
	for name, manifest := range result.Manifests {
		manifests[name] = manifest
	}

	s.logger.Info("Successfully generated manifests", "count", len(manifests))

	return &proto.RenderManifestsResponse{
		Manifests:  manifests,
		BundleData: result.Module,
	}, nil
}

// HealthCheck implements the HealthCheck gRPC method
func (s *RendererService) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	return &proto.HealthCheckResponse{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
	}, nil
}

// convertProtoBundle converts a protobuf ModuleBundle to deployment.ModuleBundle
func (s *RendererService) convertProtoBundle(pb *sp.ModuleBundle) (deployment.ModuleBundle, error) {
	// Create the bundle structure using CUE
	bundleValue := s.ctx.Encode(map[string]interface{}{
		"env":     pb.Env,
		"modules": s.convertProtoModules(pb.Modules),
	})

	if bundleValue.Err() != nil {
		return deployment.ModuleBundle{}, fmt.Errorf("failed to encode bundle: %w", bundleValue.Err())
	}

	// Parse the bundle value into the expected structure
	return deployment.ParseBundleValue(bundleValue)
}

// convertProtoModules converts protobuf modules to the expected format
func (s *RendererService) convertProtoModules(protoModules map[string]*sp.Module) map[string]interface{} {
	modules := make(map[string]interface{})
	for name, protoModule := range protoModules {
		module := map[string]interface{}{
			"instance":  protoModule.Instance,
			"name":      protoModule.Name,
			"namespace": protoModule.Namespace,
			"path":      protoModule.Path,
			"registry":  protoModule.Registry,
			"type":      protoModule.Type,
			"version":   protoModule.Version,
		}

		// Parse values if provided
		if len(protoModule.Values) > 0 {
			valuesValue := s.ctx.CompileBytes(protoModule.Values)
			if valuesValue.Err() == nil {
				module["values"] = valuesValue
			}
		}

		modules[name] = module
	}
	return modules
}