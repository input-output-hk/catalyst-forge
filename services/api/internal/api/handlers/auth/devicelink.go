package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

// DeviceLinkDeps contains dependencies for device link handlers.
type DeviceLinkDeps struct {
	DeviceLinkService service.DeviceLinkService
	UserStore         store.UserStore
	RateLimiter       rate.DeviceLinkLimiter
}

// RegisterDeviceLink registers all device-link endpoints.
func RegisterDeviceLink(r *gin.Engine, deps DeviceLinkDeps) {
	api := r.Group("/api/v1/auth/device-link")
	
	// @Summary Begin device link flow
	// @Description Initiates a device authorization flow for CLI/device authentication
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body BeginDeviceLinkRequest true "Device link request"
	// @Success 200 {object} service.DeviceLinkResponse "Device link initiated successfully"
	// @Failure 429 {object} httpkit.ErrorResponse "Rate limit exceeded"
	// @Failure 500 {object} httpkit.ErrorResponse "Internal server error"
	// @Router /api/v1/auth/device-link/begin [post]
	api.POST("/begin", beginDeviceLinkHandler(deps.DeviceLinkService, deps.RateLimiter))
	
	// @Summary Authorize device link
	// @Description Authorizes a pending device link request (requires authentication and step-up)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Security BearerAuth
	// @Param request body AuthorizeDeviceLinkRequest true "Authorization request"
	// @Success 204 "Device link authorized successfully"
	// @Failure 400 {object} httpkit.ErrorResponse "Invalid or expired code"
	// @Failure 401 {object} httpkit.ErrorResponse "Authentication required"
	// @Failure 428 {object} httpkit.ErrorResponse "Step-up authentication required"
	// @Router /api/v1/auth/device-link/authorize [post]
	api.POST("/authorize", authorizeDeviceLinkHandler(deps.DeviceLinkService))
	
	// @Summary Exchange device code for tokens
	// @Description Exchanges a device code for access and refresh tokens (polling endpoint)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body ExchangeDeviceCodeRequest true "Exchange request"
	// @Success 200 {object} service.ExchangeResponse "Tokens issued successfully"
	// @Failure 400 {object} DeviceCodeErrorResponse "Authorization pending, expired, or slow down"
	// @Failure 500 {object} httpkit.ErrorResponse "Internal server error"
	// @Router /api/v1/auth/device-link/exchange [post]
	api.POST("/exchange", exchangeDeviceCodeHandler(deps.DeviceLinkService, deps.RateLimiter))
}

// BeginDeviceLinkRequest represents the request to begin a device link flow
type BeginDeviceLinkRequest struct {
	DeviceName string `json:"device_name" example:"My CLI"`
	Purpose    string `json:"purpose" example:"login" enums:"login,step_up"`
}

// AuthorizeDeviceLinkRequest represents the request to authorize a device link
type AuthorizeDeviceLinkRequest struct {
	UserCode string `json:"user_code" example:"ABCD-123" binding:"required"`
}

// ExchangeDeviceCodeRequest represents the request to exchange a device code
type ExchangeDeviceCodeRequest struct {
	DeviceCode string `json:"device_code" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." binding:"required"`
}

// DeviceCodeErrorResponse represents error responses for device code exchange
type DeviceCodeErrorResponse struct {
	Error            string `json:"error" example:"authorization_pending"`
	ErrorDescription string `json:"error_description" example:"The authorization request is still pending"`
}

// beginDeviceLinkHandler handles POST /api/v1/auth/device-link/begin
func beginDeviceLinkHandler(linkService service.DeviceLinkService, limiter rate.DeviceLinkLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP for rate limiting
		clientIP := getClientIP(c)
		
		// Check rate limit
		if limiter != nil {
			allowed, err := limiter.AllowBegin(c.Request.Context(), clientIP)
			if err != nil || !allowed {
				httpkit.ErrorResponse(c.Writer, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
				return
			}
		}
		
		var req struct {
			DeviceName string `json:"device_name"`
			Purpose    string `json:"purpose"` // "login" or "step_up"
		}
		
		if err := httpkit.ParseJSON(c.Writer, c.Request, &req); err != nil {
			return
		}
		
		// Default device name if not provided
		if req.DeviceName == "" {
			req.DeviceName = "CLI Device"
		}
		
		// Default purpose to login
		if req.Purpose == "" {
			req.Purpose = "login"
		}
		
		// Begin the device link flow
		resp, err := linkService.BeginDeviceLink(c.Request.Context(), req.DeviceName, req.Purpose)
		if err != nil {
			_ = httpkit.NewInternalError().Write(c.Writer)
			return
		}
		
		_ = httpkit.WriteJSON(c.Writer, http.StatusOK, resp)
	}
}

// authorizeDeviceLinkHandler handles POST /api/v1/auth/device-link/authorize
func authorizeDeviceLinkHandler(linkService service.DeviceLinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check authentication
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = httpkit.NewUnauthorizedError("authentication required").Write(c.Writer)
			return
		}
		
		// Check for fresh step-up
		if ctx.RequiresStepUp(time.Now().UTC()) {
			httpkit.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Fresh authentication required")
			return
		}
		
		var req struct {
			UserCode string `json:"user_code"`
		}
		
		if err := httpkit.ParseJSON(c.Writer, c.Request, &req); err != nil {
			return
		}
		
		if req.UserCode == "" {
			_ = httpkit.NewBadRequestError("user_code is required").Write(c.Writer)
			return
		}
		
		// Authorize the device link
		err := linkService.AuthorizeDeviceLink(c.Request.Context(), req.UserCode, ctx.UserID)
		if err != nil {
			_ = httpkit.NewBadRequestError("invalid or expired code").Write(c.Writer)
			return
		}
		
		c.Status(http.StatusNoContent)
	}
}

// exchangeDeviceCodeHandler handles POST /api/v1/auth/device-link/exchange
func exchangeDeviceCodeHandler(linkService service.DeviceLinkService, limiter rate.DeviceLinkLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceCode string `json:"device_code"`
		}
		
		if err := httpkit.ParseJSON(c.Writer, c.Request, &req); err != nil {
			return
		}
		
		if req.DeviceCode == "" {
			_ = httpkit.NewBadRequestError("device_code is required").Write(c.Writer)
			return
		}
		
		// Check rate limit
		if limiter != nil {
			allowed, err := limiter.AllowExchange(c.Request.Context(), req.DeviceCode)
			if err != nil {
				// Check if it's a slow_down error
				if err.Error() == "slow_down" {
					_ = httpkit.WriteJSON(c.Writer, http.StatusBadRequest, map[string]string{
						"error": "slow_down",
						"error_description": "You are polling too frequently",
					})
					return
				}
			}
			if !allowed {
				_ = limiter.RecordSlowDown(c.Request.Context(), req.DeviceCode)
				_ = httpkit.WriteJSON(c.Writer, http.StatusBadRequest, map[string]string{
					"error": "slow_down",
					"error_description": "You are polling too frequently",
				})
				return
			}
		}
		
		// Exchange the device code
		resp, err := linkService.ExchangeDeviceCode(c.Request.Context(), req.DeviceCode)
		if err != nil {
			_ = httpkit.NewInternalError().Write(c.Writer)
			return
		}
		
		// Check status for pending/expired responses
		if resp.Status != "" {
			// Return status codes for polling clients
			switch resp.Status {
			case "authorization_pending":
				_ = httpkit.WriteJSON(c.Writer, http.StatusBadRequest, map[string]string{
					"error": "authorization_pending",
					"error_description": "The authorization request is still pending",
				})
			case "slow_down":
				_ = httpkit.WriteJSON(c.Writer, http.StatusBadRequest, map[string]string{
					"error": "slow_down",
					"error_description": "You are polling too frequently",
				})
			case "expired_token":
				_ = httpkit.WriteJSON(c.Writer, http.StatusBadRequest, map[string]string{
					"error": "expired_token",
					"error_description": "The device code has expired",
				})
			default:
				_ = httpkit.NewInternalError().Write(c.Writer)
			}
			return
		}
		
		// Success - reset rate limit and return tokens
		if limiter != nil {
			_ = limiter.ResetExchange(c.Request.Context(), req.DeviceCode)
		}
		_ = httpkit.WriteJSON(c.Writer, http.StatusOK, resp)
	}
}

// RegisterDeviceLinkVerify registers the user code verification endpoint.
// This is used by the browser UI to display device information.
func RegisterDeviceLinkVerify(r *gin.Engine, linkStore store.DeviceLinkStore) {
	// @Summary Verify device link code
	// @Description Verifies a user code and returns device information for display in the browser UI
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param code query string true "User code to verify" example:"ABCD-123"
	// @Success 200 {object} DeviceLinkVerificationResponse "Device link information"
	// @Failure 400 {object} httpkit.ErrorResponse "Invalid or expired code"
	// @Router /api/v1/auth/device-link/verify [get]
	r.GET("/api/v1/auth/device-link/verify", func(c *gin.Context) {
		userCode := c.Query("code")
		if userCode == "" {
			_ = httpkit.NewBadRequestError("code parameter is required").Write(c.Writer)
			return
		}
		
		link, err := service.VerifyDeviceCode(c.Request.Context(), linkStore, userCode)
		if err != nil {
			_ = httpkit.NewBadRequestError("invalid or expired code").Write(c.Writer)
			return
		}
		
		_ = httpkit.WriteJSON(c.Writer, http.StatusOK, map[string]interface{}{
			"device_name": link.DeviceName,
			"purpose":     link.Purpose,
			"expires_at":  link.ExpiresAt,
			"authorized":  link.AuthorizedAt != nil,
		})
	})
}

// DeviceLinkVerificationResponse represents the response for device link verification
type DeviceLinkVerificationResponse struct {
	DeviceName string    `json:"device_name" example:"My CLI"`
	Purpose    string    `json:"purpose" example:"login" enums:"login,step_up"`
	ExpiresAt  time.Time `json:"expires_at" example:"2024-01-02T15:04:05Z"`
	Authorized bool      `json:"authorized" example:"false"`
}

// getClientIP extracts the client IP address from the request.
func getClientIP(c *gin.Context) string {
	// Try X-Forwarded-For header first
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	
	// Try X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to remote address
	return c.ClientIP()
}