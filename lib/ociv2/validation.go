package ociv2

import (
	"context"
	"fmt"
	"slices"
)

// VerifyArtifact pulls the artifact, (optionally) verifies signature,
// runs shape checks, then executes any JSON validations.
// It returns the pulled artifact so callers can keep using it.
func (c *client) VerifyArtifact(ctx context.Context, ref string, opts VerifyOptions) (*PullResult, *ValidationReport, error) {
	report := &ValidationReport{
		JSONErrors: make(map[string]error),
	}
	
	// Step 1: Signature verification (if required)
	if opts.RequireSignature {
		verifyReport, err := c.VerifyDigest(ctx, ref)
		if err != nil {
			report.SignatureError = err
			return nil, report, fmt.Errorf("signature verification failed: %w", err)
		}
		
		// Check if signature is valid based on the signing report
		report.SignatureValid = len(verifyReport.Signers) > 0 && len(verifyReport.Errors) == 0
		if !report.SignatureValid {
			report.SignatureError = fmt.Errorf("signature verification failed: %v", verifyReport.Errors)
			return nil, report, report.SignatureError
		}
	} else {
		report.SignatureValid = true // Not required, so consider it valid
	}
	
	// Step 2: Pull the artifact
	pullResult, err := c.PullArtifact(ctx, ref)
	if err != nil {
		return nil, report, fmt.Errorf("failed to pull artifact: %w", err)
	}
	
	// Step 3: Shape validation (if specified)
	if opts.Shape != nil {
		if err := validateShape(pullResult, opts.Shape); err != nil {
			report.ShapeError = err
			return pullResult, report, fmt.Errorf("shape validation failed: %w", err)
		}
	}
	report.ShapeValid = true
	
	// Step 4: JSON validations
	allJSONValid := true
	for _, validation := range opts.JSONValidations {
		// Extract JSON using selector
		jsonDoc, err := validation.Selector.Extract(ctx, pullResult)
		if err != nil {
			report.JSONErrors[validation.Name] = fmt.Errorf("selector failed: %w", err)
			allJSONValid = false
			continue
		}
		
		// Validate JSON using validator
		if err := validation.Validator.Validate(ctx, jsonDoc); err != nil {
			report.JSONErrors[validation.Name] = fmt.Errorf("validation failed: %w", err)
			allJSONValid = false
			continue
		}
	}
	report.JSONValid = allJSONValid
	
	// Overall validity
	report.Valid = report.SignatureValid && report.ShapeValid && report.JSONValid
	
	return pullResult, report, nil
}

// validateShape validates the shape of a pulled artifact against the specification
func validateShape(pr *PullResult, spec *ShapeValidationSpec) error {
	// Check artifact type
	if spec.RequireArtifactType != "" && pr.ArtifactType != spec.RequireArtifactType {
		return fmt.Errorf("artifact type mismatch: expected %q, got %q", spec.RequireArtifactType, pr.ArtifactType)
	}
	
	// Check config media type
	if spec.RequireConfigMediaType != "" && pr.ConfigMediaType != spec.RequireConfigMediaType {
		return fmt.Errorf("config media type mismatch: expected %q, got %q", spec.RequireConfigMediaType, pr.ConfigMediaType)
	}
	
	// Check layer count constraints
	if spec.RequireLayerCount != nil {
		layerCount := len(pr.Layers)
		if layerCount < spec.RequireLayerCount.Min {
			return fmt.Errorf("too few layers: expected at least %d, got %d", spec.RequireLayerCount.Min, layerCount)
		}
		if layerCount > spec.RequireLayerCount.Max {
			return fmt.Errorf("too many layers: expected at most %d, got %d", spec.RequireLayerCount.Max, layerCount)
		}
	}
	
	// Check allowed layer media types
	if len(spec.AllowedLayerMediaTypes) > 0 {
		for i, layer := range pr.Layers {
			if !slices.Contains(spec.AllowedLayerMediaTypes, layer.MediaType) {
				return fmt.Errorf("layer %d has disallowed media type %q, allowed: %v", i, layer.MediaType, spec.AllowedLayerMediaTypes)
			}
		}
	}
	
	// Check required manifest annotations
	if len(spec.RequireManifestAnn) > 0 {
		for key, required := range spec.RequireManifestAnn {
			if required {
				if _, exists := pr.ManifestAnn[key]; !exists {
					return fmt.Errorf("required manifest annotation %q is missing", key)
				}
			}
		}
	}
	
	return nil
}