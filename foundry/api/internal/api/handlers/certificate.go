package handlers

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/ca"
	metrics "github.com/input-output-hk/catalyst-forge/foundry/api/internal/metrics"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/rate"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	pca "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/pca"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/utils"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
	"gorm.io/datatypes"
)

// CertificateSigningRequest represents a request to sign a certificate
type CertificateSigningRequest struct {
	// CSR is the PEM-encoded Certificate Signing Request
	CSR string `json:"csr" binding:"required" example:"-----BEGIN CERTIFICATE REQUEST-----\n..."`

	// SANs are additional Subject Alternative Names to include
	// These will be validated against user permissions. For client certs, use URI SANs.
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
	jwtManager jwt.JWTManager
	pcaClient  pca.PCAClient
	limiter    rate.Limiter
}

// NewCertificateHandler creates a new certificate handler
func NewCertificateHandler(jwtManager jwt.JWTManager) *CertificateHandler {
	return &CertificateHandler{
		jwtManager: jwtManager,
		limiter:    rate.NewInMemoryLimiter(),
	}
}

// WithPCA sets the PCA client on the handler
func (h *CertificateHandler) WithPCA(client pca.PCAClient) *CertificateHandler {
	h.pcaClient = client
	return h
}

// SignCertificate handles certificate signing requests
// @Summary Sign a certificate
// @Description Signs a Certificate Signing Request (CSR)
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
	}

	// Rate limit per principal (user ID or repo) per hour (policy key ISSUANCE_RATE_HOURLY; default 20)
	rateLimit := 20
	if v, ok := utils.GetString(c, "certs_issuance_rate_hourly"); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateLimit = n
		}
	}
	// Use subject from claims if available; else fall back to IP
	principalKey := ""
	if u, ok2 := userData.(*middleware.AuthenticatedUser); ok2 && u.Claims != nil {
		principalKey = u.Claims.Subject
	}
	if principalKey == "" {
		principalKey = c.ClientIP()
	}
	if ok, _ := h.limiter.Allow(c, "cert-issue:"+principalKey, rateLimit, time.Hour); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "certificate issuance rate limit exceeded"})
		return
	}

	user, ok := userData.(*middleware.AuthenticatedUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user data"})
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

	// Verify CSR signature & apply basic validator rules
	if err := csr.CheckSignature(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("CSR signature verification failed: %v", err)})
		return
	}

	// Parse TTL and clamp by policy
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

	// Apply CSR validator rules based on intent (client default, server when DNS/IP SANs provided)
	// If there are any DNS or IP SANs, treat as server CSR; otherwise, treat as client CSR
	if len(csr.DNSNames) > 0 || len(csr.IPAddresses) > 0 || len(sans) > 0 {
		// server/gateway issuance path
		if err := ca.ValidateServerCSR(csr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid server CSR: %v", err)})
			return
		}
	} else {
		// client (dev/ci) issuance path
		if err := ca.ValidateClientCSR(csr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid client CSR: %v", err)})
			return
		}
	}

	// Policy clamps based on certificate kind (use helpers for context reads)
	isServer := len(csr.DNSNames) > 0 || len(csr.IPAddresses) > 0
	if isServer {
		// Server clamp: default 6d if not configured
		maxServer := 6 * 24 * time.Hour
		if d, ok := utils.GetDuration(c, "certs_server_cert_ttl"); ok && d > 0 {
			maxServer = d
		}
		if ttl > maxServer {
			ttl = maxServer
		}
	} else {
		// Client clamps: dev default 90m; CI default 120m and must not exceed job token exp
		maxDev := 90 * time.Minute
		if d, ok := utils.GetDuration(c, "certs_client_cert_ttl_dev"); ok && d > 0 {
			maxDev = d
		}
		if ttl > maxDev {
			ttl = maxDev
		}
		maxCI := 120 * time.Minute
		if d, ok := utils.GetDuration(c, "certs_client_cert_ttl_ci_max"); ok && d > 0 {
			maxCI = d
		}
		if ttl > maxCI {
			ttl = maxCI
		}
		if user != nil && user.Claims != nil && user.Claims.ExpiresAt != nil {
			untilExp := time.Until(user.Claims.ExpiresAt.Time)
			if untilExp > 0 && ttl > untilExp {
				ttl = untilExp
			}
		}
	}

	// Build SANs for APIPassthrough
	pcaSANs := pca.SANs{}
	if isServer {
		pcaSANs.DNS = append(pcaSANs.DNS, csr.DNSNames...)
		pcaSANs.DNS = append(pcaSANs.DNS, sans...)
	} else {
		// client: use URI SANs from CSR only; enforce via validator already
		for _, u := range csr.URIs {
			if u != nil {
				pcaSANs.URIs = append(pcaSANs.URIs, u.String())
			}
		}
	}
	start := time.Now()
	// Choose CA/template/algo based on kind
	caArnKey, tplArnKey, algoKey := "certs_pca_client_ca_arn", "certs_pca_client_template_arn", "certs_pca_signing_algo_client"
	if isServer {
		caArnKey, tplArnKey, algoKey = "certs_pca_server_ca_arn", "certs_pca_server_template_arn", "certs_pca_signing_algo_server"
	}
	caArn, _ := utils.GetString(c, caArnKey)
	tplArn, _ := utils.GetString(c, tplArnKey)
	algo, _ := utils.GetString(c, algoKey)
	if caArn == "" {
		caArn = "arn:mock:client"
	}
	if tplArn == "" {
		tplArn = "arn:aws:acm-pca:::template/EndEntityClientAuthCertificate_APIPassthrough/V1"
	}
	if algo == "" {
		algo = "SHA256WITHECDSA"
	}
	certArn, err := h.pcaClient.Issue(c, caArn, tplArn, algo, block.Bytes, ttl, pcaSANs)
	if err != nil {
		if metrics.CertIssueErrorsTotal != nil {
			metrics.CertIssueErrorsTotal.WithLabelValues("pca_issue_error").Inc()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("pca issue failed: %v", err)})
		return
	}
	certPEM, chainPEM, err := h.pcaClient.Get(c, caArn, certArn)
	if err != nil {
		if metrics.CertIssueErrorsTotal != nil {
			metrics.CertIssueErrorsTotal.WithLabelValues("pca_get_error").Inc()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("pca get failed: %v", err)})
		return
	}
	if metrics.PCAIssueLatencySeconds != nil {
		kind := "client"
		if isServer {
			kind = "server"
		}
		metrics.PCAIssueLatencySeconds.WithLabelValues(kind).Observe(time.Since(start).Seconds())
	}
	// Reuse existing parsing logic by fabricating a SignResponse-like struct
	chain := splitPEMCerts(chainPEM)
	signResp := struct {
		Certificate string
		Chain       []string
	}{Certificate: certPEM, Chain: chain}
	// Parse the signed certificate to extract metadata
	certBlock, _ := pem.Decode([]byte(signResp.Certificate))
	if certBlock == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid certificate returned from PCA"})
		return
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to parse certificate: %v", err)})
		return
	}
	fingerprint := sha256.Sum256(cert.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])
	response := CertificateSigningResponse{
		Certificate:      signResp.Certificate,
		CertificateChain: signResp.Chain,
		SerialNumber:     cert.SerialNumber.String(),
		NotBefore:        cert.NotBefore,
		NotAfter:         cert.NotAfter,
		Fingerprint:      fmt.Sprintf("sha256:%s", fingerprintHex),
	}
	if metrics.CertIssuedTotal != nil {
		kind := "client"
		if isServer {
			kind = "server"
		}
		metrics.CertIssuedTotal.WithLabelValues(kind).Inc()
	}
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{
				EventType: "cert.issued",
				RequestIP: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Metadata:  buildAuditMetadata(subject, sans, ttl, map[string]any{"serial": cert.SerialNumber.String(), "not_after": cert.NotAfter, "ca_arn": caArn, "template_arn": tplArn, "signing_algo": algo}),
			})
		}
	}
	c.JSON(http.StatusOK, response)
	return

}

// SignServerCertificate handles BuildKit server certificate issuance via Step-CA servers provisioner
// @Summary Sign a BuildKit server certificate
// @Description Signs a server CSR
// @Tags certificates
// @Accept json
// @Produce json
// @Param request body CertificateSigningRequest true "Server certificate signing request"
// @Success 200 {object} CertificateSigningResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden - insufficient permissions"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /ca/buildkit/server-certificates [post]
// @Security BearerAuth
func (h *CertificateHandler) SignServerCertificate(c *gin.Context) {
	// Bind and parse
	var req CertificateSigningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}
	block, _ := pem.Decode([]byte(req.CSR))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CSR: must be PEM-encoded CERTIFICATE REQUEST"})
		return
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid CSR: %v", err)})
		return
	}
	if err := csr.CheckSignature(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("CSR signature verification failed: %v", err)})
		return
	}
	if err := ca.ValidateServerCSR(csr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid server CSR: %v", err)})
		return
	}

	// TTL parse + clamp
	ttl := 2 * time.Hour
	if req.TTL != "" {
		if d, err := time.ParseDuration(req.TTL); err == nil {
			ttl = d
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid TTL format: %v", err)})
			return
		}
	}
	if d, ok := utils.GetDuration(c, "certs_server_cert_ttl"); ok && d > 0 && ttl > d {
		ttl = d
	}

	pcaSANs := pca.SANs{DNS: csr.DNSNames}
	start := time.Now()
	caArn, _ := utils.GetString(c, "certs_pca_server_ca_arn")
	tplArn, _ := utils.GetString(c, "certs_pca_server_template_arn")
	algo, _ := utils.GetString(c, "certs_pca_signing_algo_server")
	if caArn == "" {
		caArn = "arn:mock:server"
	}
	if tplArn == "" {
		tplArn = "arn:aws:acm-pca:::template/EndEntityServerAuthCertificate_APIPassthrough/V1"
	}
	if algo == "" {
		algo = "SHA256WITHECDSA"
	}
	certArn, err := h.pcaClient.Issue(c, caArn, tplArn, algo, block.Bytes, ttl, pcaSANs)
	if err != nil {
		if metrics.CertIssueErrorsTotal != nil {
			metrics.CertIssueErrorsTotal.WithLabelValues("pca_issue_error").Inc()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("pca issue failed: %v", err)})
		return
	}
	certPEM, chainPEM, err := h.pcaClient.Get(c, caArn, certArn)
	if err != nil {
		if metrics.CertIssueErrorsTotal != nil {
			metrics.CertIssueErrorsTotal.WithLabelValues("pca_get_error").Inc()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("pca get failed: %v", err)})
		return
	}
	if metrics.PCAIssueLatencySeconds != nil {
		metrics.PCAIssueLatencySeconds.WithLabelValues("server").Observe(time.Since(start).Seconds())
	}
	chain := splitPEMCerts(chainPEM)
	signResp := struct {
		Certificate string
		Chain       []string
	}{Certificate: certPEM, Chain: chain}
	// Parse returned certificate for fingerprint
	certBlock, _ := pem.Decode([]byte(signResp.Certificate))
	if certBlock == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid certificate returned from PCA"})
		return
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to parse certificate: %v", err)})
		return
	}
	fp := sha256.Sum256(cert.Raw)
	fingerprintHex := hex.EncodeToString(fp[:])
	resp := CertificateSigningResponse{
		Certificate:      signResp.Certificate,
		CertificateChain: signResp.Chain,
		SerialNumber:     cert.SerialNumber.String(),
		NotBefore:        cert.NotBefore,
		NotAfter:         cert.NotAfter,
		Fingerprint:      fmt.Sprintf("sha256:%s", fingerprintHex),
	}
	if metrics.CertIssuedTotal != nil {
		metrics.CertIssuedTotal.WithLabelValues("server").Inc()
	}
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{
				EventType: "servercert.issued",
				RequestIP: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Metadata:  buildAuditMetadata(csr.Subject.CommonName, csr.DNSNames, ttl, map[string]any{"serial": cert.SerialNumber.String(), "not_after": cert.NotAfter, "ca_arn": caArn, "template_arn": tplArn, "signing_algo": algo}),
			})
		}
	}
	c.JSON(http.StatusOK, resp)
	return
}

// buildAuditMetadata constructs datatypes.JSON with core cert details and extras
func buildAuditMetadata(subject string, sans []string, ttl time.Duration, extras map[string]any) datatypes.JSON {
	m := map[string]any{
		"subject": subject,
		"sans":    sans,
		"ttl":     ttl.String(),
	}
	for k, v := range extras {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return datatypes.JSON(b)
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
// GetRootCertificate returns the PEM-encoded root CA certificate used for signing.
// In local/mock mode, this fetches from the PCA mock without requiring ARNs.
func (h *CertificateHandler) GetRootCertificate(c *gin.Context) {
	// Try PCA if configured
	if h.pcaClient != nil {
		// For local/mock PCA, return a generated CA unconditionally
		if pem, _, err := h.pcaClient.GetCA(c, "arn:mock:server"); err == nil {
			c.Data(http.StatusOK, "application/x-pem-file", []byte(pem))
			return
		}
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get root certificate: PCA not configured"})
}

// splitPEMCerts splits a PEM bundle into a slice of certificate PEMs
func splitPEMCerts(bundle string) []string {
	var out []string
	data := []byte(bundle)
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			out = append(out, string(pem.EncodeToMemory(block)))
		}
	}
	return out
}
