package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDeviceLinkService implements service.DeviceLinkService for testing
type mockDeviceLinkService struct {
	beginFunc     func(ctx context.Context, deviceName string, purpose string) (*service.DeviceLinkResponse, error)
	authorizeFunc func(ctx context.Context, userCode string, userID uuid.UUID) error
	exchangeFunc  func(ctx context.Context, deviceCode string) (*service.ExchangeResponse, error)
}

func (m *mockDeviceLinkService) BeginDeviceLink(ctx context.Context, deviceName string, purpose string) (*service.DeviceLinkResponse, error) {
	if m.beginFunc != nil {
		return m.beginFunc(ctx, deviceName, purpose)
	}
	return &service.DeviceLinkResponse{
		DeviceCode:              "test-device-code",
		UserCode:                "TEST-123",
		VerificationURI:         "https://example.com/cli/link",
		VerificationURIComplete: "https://example.com/cli/link?c=TEST-123",
		ExpiresIn:               600,
		Interval:                5,
	}, nil
}

func (m *mockDeviceLinkService) AuthorizeDeviceLink(ctx context.Context, userCode string, userID uuid.UUID) error {
	if m.authorizeFunc != nil {
		return m.authorizeFunc(ctx, userCode, userID)
	}
	return nil
}

func (m *mockDeviceLinkService) ExchangeDeviceCode(ctx context.Context, deviceCode string) (*service.ExchangeResponse, error) {
	if m.exchangeFunc != nil {
		return m.exchangeFunc(ctx, deviceCode)
	}
	return &service.ExchangeResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		DeviceID:     uuid.New().String(),
	}, nil
}

// mockDeviceLinkLimiter implements rate.DeviceLinkLimiter for testing
type mockDeviceLinkLimiter struct {
	allowBeginFunc     func(ctx context.Context, ip string) (bool, error)
	allowExchangeFunc  func(ctx context.Context, deviceCode string) (bool, error)
	recordSlowDownFunc func(ctx context.Context, deviceCode string) error
	resetExchangeFunc  func(ctx context.Context, deviceCode string) error
}

func (m *mockDeviceLinkLimiter) AllowBegin(ctx context.Context, ip string) (bool, error) {
	if m.allowBeginFunc != nil {
		return m.allowBeginFunc(ctx, ip)
	}
	return true, nil
}

func (m *mockDeviceLinkLimiter) AllowExchange(ctx context.Context, deviceCode string) (bool, error) {
	if m.allowExchangeFunc != nil {
		return m.allowExchangeFunc(ctx, deviceCode)
	}
	return true, nil
}

func (m *mockDeviceLinkLimiter) RecordSlowDown(ctx context.Context, deviceCode string) error {
	if m.recordSlowDownFunc != nil {
		return m.recordSlowDownFunc(ctx, deviceCode)
	}
	return nil
}

func (m *mockDeviceLinkLimiter) ResetExchange(ctx context.Context, deviceCode string) error {
	if m.resetExchangeFunc != nil {
		return m.resetExchangeFunc(ctx, deviceCode)
	}
	return nil
}

func TestBeginDeviceLinkHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		requestBody       string
		rateLimitAllowed  bool
		serviceError      error
		wantStatus        int
		wantDeviceCode    string
		wantUserCode      string
		wantErrorKey      string
	}{
		{
			name:             "ok/successful_begin",
			requestBody:      `{"device_name":"Test CLI","purpose":"login"}`,
			rateLimitAllowed: true,
			wantStatus:       http.StatusOK,
			wantDeviceCode:   "test-device-code",
			wantUserCode:     "TEST-123",
		},
		{
			name:             "ok/default_values",
			requestBody:      `{}`,
			rateLimitAllowed: true,
			wantStatus:       http.StatusOK,
			wantDeviceCode:   "test-device-code",
			wantUserCode:     "TEST-123",
		},
		{
			name:             "error/rate_limited",
			requestBody:      `{"device_name":"Test"}`,
			rateLimitAllowed: false,
			wantStatus:       http.StatusTooManyRequests,
			wantErrorKey:     "rate_limit_exceeded",
		},
		{
			name:             "error/service_error",
			requestBody:      `{"device_name":"Test"}`,
			rateLimitAllowed: true,
			serviceError:     errors.New("internal error"),
			wantStatus:       http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			
			linkService := &mockDeviceLinkService{
				beginFunc: func(ctx context.Context, deviceName string, purpose string) (*service.DeviceLinkResponse, error) {
					if tc.serviceError != nil {
						return nil, tc.serviceError
					}
					return &service.DeviceLinkResponse{
						DeviceCode:              tc.wantDeviceCode,
						UserCode:                tc.wantUserCode,
						VerificationURI:         "https://example.com/cli/link",
						VerificationURIComplete: "https://example.com/cli/link?c=" + tc.wantUserCode,
						ExpiresIn:               600,
						Interval:                5,
					}, nil
				},
			}
			
			limiter := &mockDeviceLinkLimiter{
				allowBeginFunc: func(ctx context.Context, ip string) (bool, error) {
					return tc.rateLimitAllowed, nil
				},
			}
			
			handler := beginDeviceLinkHandler(linkService, limiter)
			r.POST("/begin", handler)

			reqBody := bytes.NewBufferString(tc.requestBody)
			req := httptest.NewRequest("POST", "/begin", reqBody)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code, "unexpected status code")

			if tc.wantStatus == http.StatusOK {
				var resp service.DeviceLinkResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err, "failed to unmarshal response")
				assert.Equal(t, tc.wantDeviceCode, resp.DeviceCode, "device code mismatch")
				assert.Equal(t, tc.wantUserCode, resp.UserCode, "user code mismatch")
				assert.Equal(t, 600, resp.ExpiresIn, "expires_in should be 600")
				assert.Equal(t, 5, resp.Interval, "interval should be 5")
			}

			if tc.wantErrorKey != "" {
				var resp map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err, "failed to unmarshal error response")
				assert.Equal(t, tc.wantErrorKey, resp["error"], "error key mismatch")
			}
		})
	}
}

func TestAuthorizeDeviceLinkHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()

	t.Run("ok/successful_authorize", func(t *testing.T) {
		r := gin.New()
		
		// Add authenticated context with valid step-up
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:           userID,
				Email:            "user@example.com",
				StepUpValidUntil: time.Now().Add(time.Hour), // Valid step-up
			}
			authCtx.Set(c)
			c.Next()
		})

		linkService := &mockDeviceLinkService{}
		handler := authorizeDeviceLinkHandler(linkService)
		r.POST("/authorize", handler)

		reqBody := bytes.NewBufferString(`{"user_code":"TEST-123"}`)
		req := httptest.NewRequest("POST", "/authorize", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("error/not_authenticated", func(t *testing.T) {
		r := gin.New()
		linkService := &mockDeviceLinkService{}
		handler := authorizeDeviceLinkHandler(linkService)
		r.POST("/authorize", handler)

		reqBody := bytes.NewBufferString(`{"user_code":"TEST-123"}`)
		req := httptest.NewRequest("POST", "/authorize", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error/requires_step_up", func(t *testing.T) {
		r := gin.New()
		
		// Add authenticated context without step-up
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:           userID,
				Email:            "user@example.com",
				StepUpValidUntil: time.Now().Add(-time.Hour), // Expired step-up
			}
			authCtx.Set(c)
			c.Next()
		})

		linkService := &mockDeviceLinkService{}
		handler := authorizeDeviceLinkHandler(linkService)
		r.POST("/authorize", handler)

		reqBody := bytes.NewBufferString(`{"user_code":"TEST-123"}`)
		req := httptest.NewRequest("POST", "/authorize", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusPreconditionRequired, w.Code)
	})

	t.Run("error/missing_user_code", func(t *testing.T) {
		r := gin.New()
		
		// Add authenticated context with valid step-up
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:           userID,
				Email:            "user@example.com",
				StepUpValidUntil: time.Now().Add(time.Hour), // Valid step-up
			}
			authCtx.Set(c)
			c.Next()
		})

		linkService := &mockDeviceLinkService{}
		handler := authorizeDeviceLinkHandler(linkService)
		r.POST("/authorize", handler)

		reqBody := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest("POST", "/authorize", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error/invalid_code", func(t *testing.T) {
		r := gin.New()
		
		// Add authenticated context with valid step-up
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:           userID,
				Email:            "user@example.com",
				StepUpValidUntil: time.Now().Add(time.Hour), // Valid step-up
			}
			authCtx.Set(c)
			c.Next()
		})

		linkService := &mockDeviceLinkService{
			authorizeFunc: func(ctx context.Context, userCode string, userID uuid.UUID) error {
				return errors.New("invalid or expired code")
			},
		}
		handler := authorizeDeviceLinkHandler(linkService)
		r.POST("/authorize", handler)

		reqBody := bytes.NewBufferString(`{"user_code":"INVALID"}`)
		req := httptest.NewRequest("POST", "/authorize", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestExchangeDeviceCodeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ok/successful_exchange", func(t *testing.T) {
		r := gin.New()
		linkService := &mockDeviceLinkService{
			exchangeFunc: func(ctx context.Context, deviceCode string) (*service.ExchangeResponse, error) {
				return &service.ExchangeResponse{
					AccessToken:      "access-token",
					RefreshToken:     "refresh-token",
					ExpiresIn:        3600,
					RefreshExpiresIn: 604800,
					DeviceID:         uuid.New().String(),
					User: &service.UserInfo{
						ID:    uuid.New().String(),
						Email: "user@example.com",
						Roles: []string{"user"},
					},
				}, nil
			},
		}
		limiter := &mockDeviceLinkLimiter{}
		handler := exchangeDeviceCodeHandler(linkService, limiter)
		r.POST("/exchange", handler)

		reqBody := bytes.NewBufferString(`{"device_code":"test-device-code"}`)
		req := httptest.NewRequest("POST", "/exchange", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp service.ExchangeResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.Equal(t, "refresh-token", resp.RefreshToken)
		assert.Equal(t, 3600, resp.ExpiresIn)
		assert.NotEmpty(t, resp.DeviceID)
	})

	t.Run("ok/authorization_pending", func(t *testing.T) {
		r := gin.New()
		linkService := &mockDeviceLinkService{
			exchangeFunc: func(ctx context.Context, deviceCode string) (*service.ExchangeResponse, error) {
				return &service.ExchangeResponse{
					Status: "authorization_pending",
				}, nil
			},
		}
		limiter := &mockDeviceLinkLimiter{}
		handler := exchangeDeviceCodeHandler(linkService, limiter)
		r.POST("/exchange", handler)

		reqBody := bytes.NewBufferString(`{"device_code":"test-device-code"}`)
		req := httptest.NewRequest("POST", "/exchange", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "authorization_pending", resp["error"])
	})

	t.Run("ok/expired_token", func(t *testing.T) {
		r := gin.New()
		linkService := &mockDeviceLinkService{
			exchangeFunc: func(ctx context.Context, deviceCode string) (*service.ExchangeResponse, error) {
				return &service.ExchangeResponse{
					Status: "expired_token",
				}, nil
			},
		}
		limiter := &mockDeviceLinkLimiter{}
		handler := exchangeDeviceCodeHandler(linkService, limiter)
		r.POST("/exchange", handler)

		reqBody := bytes.NewBufferString(`{"device_code":"expired-code"}`)
		req := httptest.NewRequest("POST", "/exchange", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "expired_token", resp["error"])
	})

	t.Run("error/slow_down", func(t *testing.T) {
		r := gin.New()
		linkService := &mockDeviceLinkService{}
		limiter := &mockDeviceLinkLimiter{
			allowExchangeFunc: func(ctx context.Context, deviceCode string) (bool, error) {
				return false, errors.New("slow_down")
			},
		}
		handler := exchangeDeviceCodeHandler(linkService, limiter)
		r.POST("/exchange", handler)

		reqBody := bytes.NewBufferString(`{"device_code":"test-code"}`)
		req := httptest.NewRequest("POST", "/exchange", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "slow_down", resp["error"])
	})

	t.Run("error/missing_device_code", func(t *testing.T) {
		r := gin.New()
		linkService := &mockDeviceLinkService{}
		limiter := &mockDeviceLinkLimiter{}
		handler := exchangeDeviceCodeHandler(linkService, limiter)
		r.POST("/exchange", handler)

		reqBody := bytes.NewBufferString(`{}`)
		req := httptest.NewRequest("POST", "/exchange", reqBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRegisterDeviceLinkVerify(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ok/valid_code", func(t *testing.T) {
		r := gin.New()
		
		// Mock store
		linkStore := &mockDeviceLinkStore{
			getByUserCodeFunc: func(ctx context.Context, userCode string) (*domain.DeviceLink, error) {
				return &domain.DeviceLink{
					ID:         uuid.New(),
					UserCode:   userCode,
					DeviceName: "Test Device",
					Purpose:    "login",
					ExpiresAt:  time.Now().Add(time.Hour),
				}, nil
			},
		}
		
		RegisterDeviceLinkVerify(r, linkStore)

		req := httptest.NewRequest("GET", "/api/v1/auth/device-link/verify?code=TEST-123", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Test Device", resp["device_name"])
		assert.Equal(t, "login", resp["purpose"])
		assert.Equal(t, false, resp["authorized"])
	})

	t.Run("error/missing_code", func(t *testing.T) {
		r := gin.New()
		linkStore := &mockDeviceLinkStore{}
		RegisterDeviceLinkVerify(r, linkStore)

		req := httptest.NewRequest("GET", "/api/v1/auth/device-link/verify", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error/invalid_code", func(t *testing.T) {
		r := gin.New()
		linkStore := &mockDeviceLinkStore{
			getByUserCodeFunc: func(ctx context.Context, userCode string) (*domain.DeviceLink, error) {
				return nil, nil
			},
		}
		RegisterDeviceLinkVerify(r, linkStore)

		req := httptest.NewRequest("GET", "/api/v1/auth/device-link/verify?code=INVALID", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// mockDeviceLinkStore implements store.DeviceLinkStore for testing
type mockDeviceLinkStore struct {
	getByUserCodeFunc func(ctx context.Context, userCode string) (*domain.DeviceLink, error)
}

func (m *mockDeviceLinkStore) Create(ctx context.Context, link *domain.DeviceLink) error {
	return nil
}

func (m *mockDeviceLinkStore) GetByDeviceCode(ctx context.Context, deviceCodeHash []byte) (*domain.DeviceLink, error) {
	return nil, nil
}

func (m *mockDeviceLinkStore) GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceLink, error) {
	if m.getByUserCodeFunc != nil {
		return m.getByUserCodeFunc(ctx, userCode)
	}
	return nil, nil
}

func (m *mockDeviceLinkStore) Authorize(ctx context.Context, id uuid.UUID, userID uuid.UUID, authorizedAt time.Time) error {
	return nil
}

func (m *mockDeviceLinkStore) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockDeviceLinkStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}