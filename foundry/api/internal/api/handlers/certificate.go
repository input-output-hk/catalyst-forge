package handlers

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/stepca"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
)

// CertificateSigningRequest represents a request to sign a certificate
type CertificateSigningRequest struct {
	// CSR is the PEM-encoded Certificate Signing Request
	CSR string `json:"csr" binding:"required" example:"-----BEGIN CERTIFICATE REQUEST-----\n..."`

	// SANs are additional Subject Alternative Names to include
	// These will be validated against user permissions
	SANs []string `json:"sans,omitempty" example:"example.com,*.example.com"`

	// CommonName can override the CN in the CSR
	CommonName string `json:"common_name,omitempty" example:"user.example.com"`

	// TTL is the requested certificate lifetime
	// Will be capped by server policy
	TTL string `json:"ttl,omitempty" example:"24h"`
}

// CertificateSigningResponse represents the response after signing a certificate
type CertificateSigningResponse struct {
	// Certificate is the PEM-encoded signed certificate
	Certificate string `json:"certificate" example:"-----BEGIN CERTIFICATE-----\n..."`

	// CertificateChain includes intermediate certificates if available
	CertificateChain []string `json:"certificate_chain,omitempty"`

	// SerialNumber is the certificate's serial number
	SerialNumber string `json:"serial_number" example:"123456789"`

	// NotBefore is when the certificate becomes valid
	NotBefore time.Time `json:"not_before" example:"2024-01-01T00:00:00Z"`

	// NotAfter is when the certificate expires
	NotAfter time.Time `json:"not_after" example:"2024-01-02T00:00:00Z"`

	// Fingerprint is the SHA256 fingerprint of the certificate
	Fingerprint string `json:"fingerprint" example:"sha256:abcdef..."`
}

// CertificateHandler handles certificate-related API endpoints
type CertificateHandler struct {
	jwtManager   jwt.JWTManager
	stepCAClient stepca.StepCAClient
}

// NewCertificateHandler creates a new certificate handler
func NewCertificateHandler(jwtManager jwt.JWTManager, stepCAClient stepca.StepCAClient) *CertificateHandler {
	return &CertificateHandler{
		jwtManager:   jwtManager,
		stepCAClient: stepCAClient,
	}
}

// SignCertificate handles certificate signing requests
// @Summary Sign a certificate
// @Description Signs a Certificate Signing Request (CSR) using step-ca
// @Tags certificates
// @Accept json
// @Produce json
// @Param request body CertificateSigningRequest true "Certificate signing request"
// @Success 200 {object} CertificateSigningResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden - insufficient permissions"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /certificates/sign [post]
// @Security BearerAuth
func (h *CertificateHandler) SignCertificate(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	user, ok := userData.(*middleware.AuthenticatedUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid user data",
		})
		return
	}

	// Parse request
	var req CertificateSigningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Validate and parse CSR
	csrPEM := []byte(req.CSR)
	block, _ := pem.Decode(csrPEM)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid CSR: must be PEM-encoded CERTIFICATE REQUEST",
		})
		return
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid CSR: %v", err),
		})
		return
	}

	// Verify CSR signature
	if err := csr.CheckSignature(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("CSR signature verification failed: %v", err),
		})
		return
	}

	// Parse TTL
	ttl := 2 * time.Hour // Default TTL
	if req.TTL != "" {
		parsedTTL, err := time.ParseDuration(req.TTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid TTL format: %v", err),
			})
			return
		}
		ttl = parsedTTL
	}

	// Prepare certificate subject and SANs
	subject := csr.Subject.CommonName
	if req.CommonName != "" {
		subject = req.CommonName
	}

	// Use email from claims if no subject specified
	if subject == "" && user.Claims.Subject != "" {
		subject = user.Claims.Subject
	}

	// Combine CSR SANs with request SANs
	allSANs := append(csr.DNSNames, req.SANs...)

	// Deduplicate while preserving order
	seen := make(map[string]struct{}, len(allSANs))
	sans := make([]string, 0, len(allSANs))

	for _, s := range allSANs {
		if _, ok := seen[s]; ok {
			continue // already added
		}
		seen[s] = struct{}{}
		sans = append(sans, s)
	}

	// Validate SANs against user permissions
	if !h.validateSANs(user.Claims, sans) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "not authorized for requested SANs",
		})
		return
	}

	// Generate JWT token for step-ca
	certToken, err := tokens.GenerateCertificateSigningToken(
		h.jwtManager,
		subject,
		sans,
		csrPEM,
		tokens.WithTTL(ttl),
		tokens.WithEmail(user.Claims.Subject),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to generate certificate token: %v", err),
		})
		return
	}

	// Send request to step-ca with the requested TTL
	certResp, err := h.stepCAClient.SignCertificate(certToken, csrPEM, ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to sign certificate: %v", err),
		})
		return
	}

	// Parse the signed certificate to extract metadata
	certBlock, _ := pem.Decode([]byte(certResp.CRT))
	if certBlock == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid certificate returned from CA",
		})
		return
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to parse certificate: %v", err),
		})
		return
	}

	// Calculate fingerprint
	fingerprint := sha256.Sum256(cert.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	chain := certResp.X509Chain
	if len(chain) == 0 && certResp.CA != "" {
		chain = []string{certResp.CA}
	}

	// Build response
	response := CertificateSigningResponse{
		Certificate:      certResp.CRT,
		CertificateChain: chain,
		SerialNumber:     cert.SerialNumber.String(),
		NotBefore:        cert.NotBefore,
		NotAfter:         cert.NotAfter,
		Fingerprint:      fmt.Sprintf("sha256:%s", fingerprintHex),
	}

	c.JSON(http.StatusOK, response)
}

// validateSANs checks if the user is authorized for the requested SANs
func (h *CertificateHandler) validateSANs(claims *tokens.AuthClaims, sans []string) bool {
	// Get all certificate signing permissions for this user
	certPerms := tokens.GetCertificateSignPermissions(claims)
	if len(certPerms) == 0 {
		return false
	}

	// Check each requested SAN against user's permissions
	for _, san := range sans {
		if !h.isAuthorizedForSAN(san, certPerms) {
			return false
		}
	}
	return true
}

// isAuthorizedForSAN checks if a single SAN is authorized by any of the user's certificate permissions
func (h *CertificateHandler) isAuthorizedForSAN(san string, permissions []auth.Permission) bool {
	for _, perm := range permissions {
		if pattern, ok := auth.ParseCertificateSignPermission(perm); ok {
			if auth.MatchesDomainPattern(san, pattern) {
				return true
			}
		}
	}
	return false
}

// GetRootCertificate returns the CA's root certificate
// @Summary Get root certificate
// @Description Returns the Certificate Authority's root certificate
// @Tags certificates
// @Produce plain
// @Success 200 {string} string "PEM-encoded root certificate"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /certificates/root [get]
func (h *CertificateHandler) GetRootCertificate(c *gin.Context) {
	rootCert, err := h.stepCAClient.GetRootCertificate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get root certificate: %v", err),
		})
		return
	}

	c.Data(http.StatusOK, "application/x-pem-file", rootCert)
}
