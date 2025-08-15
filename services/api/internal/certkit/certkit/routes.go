package certkit

import (
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	authctx "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/certkit/service"
)

type signReq struct {
	CSRPEM           string `json:"csr_pem"`
	Template         string `json:"template"`
	SigningAlgorithm string `json:"signing_algorithm"`
	TTL              string `json:"ttl"`
	IdempotencyKey   string `json:"idempotency_key"`
}

func registerRoutes(rg *gin.RouterGroup, cfg Config, deps Deps) {
	// @Summary Issue a certificate via AWS PCA
	// @Description Validates the CSR and SAN policy, issues through PCA, waits until issued, and returns certificate PEMs.
	// @Tags pki
	// @Accept json
	// @Produce json
	// @Param body body signReq true "CSR request"
	// @Success 200 {object} map[string]string "certificate_pem, certificate_chain_pem, certificate_arn"
	// @Failure 400 {object} map[string]any "invalid CSR"
	// @Failure 403 {object} map[string]any "forbidden (RBAC/SAN policy)"
	// @Failure 429 {object} map[string]any "rate limited"
	// @Failure 504 {object} map[string]any "issuance timeout"
	// @Router /pki/sign [post]
	rg.POST("/sign", func(c *gin.Context) {
		var in signReq
		if err := c.ShouldBindJSON(&in); err != nil {
			_ = basehttp.NewBadRequestError("").Write(c.Writer)
			return
		}
		// Parse TTL (unused in stub; service will use it)
		if in.TTL != "" {
			if _, err := time.ParseDuration(in.TTL); err != nil {
				_ = basehttp.NewBadRequestError("invalid ttl").Write(c.Writer)
				return
			}
		}
		// Construct issuer and perform issuance
		clk := deps.Clock
		if clk == nil {
			clk = sysClock{}
		}
		iss := &service.Issuer{
			CAArn:            cfg.CAArn,
			PCA:              pcaAdapter{inner: deps.PCA},
			RBAC:             deps.RBAC,
			Clock:            clk,
			AllowedTemplates: cfg.AllowedTemplates,
			MaxTTL:           cfg.MaxTTL,
			PollInterval:     cfg.PollInterval,
			MaxWait:          cfg.MaxWait,
		}
		// Requestor from auth context if present
		reqBy := ""
		if ac, ok := authctx.From(c); ok && ac.IsAuthenticated() {
			reqBy = ac.UserID.String()
		}
		// Rate limit if limiter present
		if deps.Limiter != nil {
			key := rate.Key("pki:sign:" + reqBy)
			if ok, _, reset, _ := deps.Limiter.Allow(c.Request.Context(), key, 1, time.Minute); !ok {
				retry := int(reset.Sub(time.Now().UTC()).Seconds())
				if retry < 0 {
					retry = 0
				}
				_ = basehttp.NewRateLimitError(retry).Write(c.Writer)
				return
			}
		}

		ttl := cfg.DefaultTTL
		if in.TTL != "" {
			if d, err := time.ParseDuration(in.TTL); err == nil {
				ttl = d
			}
		}
		res, err := iss.SignCSR(c.Request.Context(), service.SignRequest{
			CSRPEM:           []byte(in.CSRPEM),
			TemplateKey:      in.Template,
			SigningAlgorithm: in.SigningAlgorithm,
			TTL:              ttl,
			IdempotencyKey:   in.IdempotencyKey,
			Requestor:        reqBy,
		})
		if err != nil {
			switch err {
			case service.ErrCSRInvalid, service.ErrCSRSignature:
				_ = basehttp.NewBadRequestError("").Write(c.Writer)
				return
			case service.ErrCSRPolicyViolation:
				_ = basehttp.NewForbiddenError("").Write(c.Writer)
				return
			case service.ErrIssuanceTimeout:
				_ = basehttp.WriteJSON(c.Writer, http.StatusGatewayTimeout, map[string]any{"error": "timeout", "message": "issuance timed out"})
				return
			default:
				_ = basehttp.NewInternalError().Write(c.Writer)
				return
			}
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]string{
			"certificate_pem":       res.CertificatePEM,
			"certificate_chain_pem": res.ChainPEM,
			"certificate_arn":       res.CertificateArn,
		})
	})

	// @Summary Get PCA CA certificate
	// @Description Returns the configured PCA CA certificate and certificate chain in PEM format.
	// @Tags pki
	// @Produce json
	// @Success 200 {object} map[string]string "ca_pem, chain_pem"
	// @Router /pki/ca [get]
	rg.GET("/ca", func(c *gin.Context) {
		ca, chain, err := deps.PCA.GetCACertificate(c.Request.Context(), cfg.CAArn)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]string{
			"ca_pem":    ca,
			"chain_pem": chain,
		})
	})
}
