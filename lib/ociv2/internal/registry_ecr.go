package internal

import (
	"strings"
	"time"
)

// ECRSpecificOptions contains ECR-specific configuration
type ECRSpecificOptions struct {
	// ForceImageManifest forces the use of image manifests for ECR
	ForceImageManifest bool
	// ExtendedTimeout uses longer timeouts for ECR operations
	ExtendedTimeout time.Duration
}

// IsECRRegistry checks if the registry is an ECR registry
func IsECRRegistry(registry string) bool {
	return strings.Contains(strings.ToLower(registry), "ecr") && 
		   strings.Contains(strings.ToLower(registry), "amazonaws.com")
}

// GetECRRegion extracts the AWS region from ECR registry hostname
func GetECRRegion(registry string) string {
	// ECR format: <account-id>.dkr.ecr.<region>.amazonaws.com
	parts := strings.Split(registry, ".")
	for i, part := range parts {
		if part == "ecr" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "us-east-1" // default fallback
}

// OptimizeForECR returns ECR-optimized settings
func OptimizeForECR(registry string) ECRSpecificOptions {
	region := GetECRRegion(registry)
	
	// ECR in some regions has better artifact support than others
	goodArtifactRegions := map[string]bool{
		"us-east-1": true,
		"us-west-2": true,
		"eu-west-1": true,
		"eu-central-1": true,
	}
	
	forceImage := !goodArtifactRegions[region]
	
	// ECR can be slower, especially for cross-region operations
	extendedTimeout := 5 * time.Minute
	
	return ECRSpecificOptions{
		ForceImageManifest: forceImage,
		ExtendedTimeout:    extendedTimeout,
	}
}

// IsECRManifestError checks if an error is specifically related to ECR manifest issues
func IsECRManifestError(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// ECR-specific error patterns
	ecrPatterns := []string{
		"manifest blob unknown",
		"unsupported manifest media type",
		"invalid image manifest",
		"manifest schema version not supported",
		"unsupported image manifest schema version",
		"repository does not exist",
		"image does not exist",
		"tag does not exist",
		"manifest unknown",
	}
	
	for _, pattern := range ecrPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	
	return false
}

// ECRAuthHelper provides ECR-specific authentication assistance
type ECRAuthHelper struct {
	Region    string
	AccountID string
}

// NewECRAuthHelper creates a new ECR auth helper from registry hostname
func NewECRAuthHelper(registry string) *ECRAuthHelper {
	// Extract account ID and region from ECR registry URL
	// Format: <account-id>.dkr.ecr.<region>.amazonaws.com
	parts := strings.Split(registry, ".")
	
	var accountID, region string
	
	if len(parts) >= 5 && parts[1] == "dkr" && parts[2] == "ecr" {
		accountID = parts[0]
		region = parts[3]
	}
	
	return &ECRAuthHelper{
		Region:    region,
		AccountID: accountID,
	}
}

// ShouldUseECRCredentialHelper determines if ECR credential helper should be used
func (h *ECRAuthHelper) ShouldUseECRCredentialHelper() bool {
	// Use ECR credential helper if we have valid region and account
	return h.Region != "" && h.AccountID != ""
}

// GetECREndpoint returns the ECR API endpoint for the region
func (h *ECRAuthHelper) GetECREndpoint() string {
	if h.Region == "" {
		return "https://ecr.us-east-1.amazonaws.com"
	}
	return "https://ecr." + h.Region + ".amazonaws.com"
}