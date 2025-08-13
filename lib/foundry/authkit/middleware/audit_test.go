package middleware_test

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/middleware"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/testing/inmemory"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogger_LogAuthEvents(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		withAuth       bool
		expectedEvent  domain.EventType
		expectNoEvent  bool
	}{
		{
			name:          "successful login",
			method:        "POST",
			path:          "/auth/login/complete",
			statusCode:    200,
			withAuth:      false,
			expectedEvent: domain.EventLoginSuccess,
		},
		{
			name:          "failed login",
			method:        "POST",
			path:          "/auth/login/complete",
			statusCode:    401,
			withAuth:      false,
			expectedEvent: domain.EventLoginFailed,
		},
		{
			name:          "login begin",
			method:        "POST",
			path:          "/auth/login/begin",
			statusCode:    200,
			withAuth:      false,
			expectedEvent: domain.EventLoginBegin,
		},
		{
			name:          "logout",
			method:        "POST",
			path:          "/auth/logout",
			statusCode:    204,
			withAuth:      true,
			expectedEvent: domain.EventLogout,
		},
		{
			name:          "logout all",
			method:        "POST",
			path:          "/auth/logout-all",
			statusCode:    204,
			withAuth:      true,
			expectedEvent: domain.EventLogoutAll,
		},
		{
			name:          "registration begin",
			method:        "POST",
			path:          "/auth/onboard/begin",
			statusCode:    200,
			withAuth:      false,
			expectedEvent: domain.EventRegistrationBegin,
		},
		{
			name:          "registration success",
			method:        "POST",
			path:          "/auth/onboard/complete",
			statusCode:    201,
			withAuth:      false,
			expectedEvent: domain.EventRegistrationSuccess,
		},
		{
			name:          "registration failed",
			method:        "POST",
			path:          "/auth/onboard/complete",
			statusCode:    400,
			withAuth:      false,
			expectedEvent: domain.EventRegistrationFailed,
		},
		{
			name:          "token refresh success",
			method:        "POST",
			path:          "/auth/refresh",
			statusCode:    200,
			withAuth:      false,
			expectedEvent: domain.EventTokenRefresh,
		},
		{
			name:          "token refresh failed",
			method:        "POST",
			path:          "/auth/refresh",
			statusCode:    401,
			withAuth:      false,
			expectedEvent: domain.EventTokenRefreshFailed,
		},
		{
			name:          "CSRF violation on refresh",
			method:        "POST",
			path:          "/auth/refresh",
			statusCode:    403,
			withAuth:      false,
			expectedEvent: domain.EventCSRFViolation,
		},
		{
			name:          "rate limit exceeded",
			method:        "POST",
			path:          "/auth/login/complete",
			statusCode:    429,
			withAuth:      false,
			expectedEvent: domain.EventRateLimitExceeded,
		},
		{
			name:          "access denied",
			method:        "POST",
			path:          "/api/admin/users",
			statusCode:    403,
			withAuth:      true,
			expectedEvent: domain.EventAccessDenied,
		},
		{
			name:          "invite created",
			method:        "POST",
			path:          "/auth/invites",
			statusCode:    201,
			withAuth:      true,
			expectedEvent: domain.EventInviteCreated,
		},
		{
			name:          "credential added",
			method:        "POST",
			path:          "/auth/credentials/add/complete",
			statusCode:    201,
			withAuth:      true,
			expectedEvent: domain.EventCredentialAdded,
		},
		{
			name:          "non-auth endpoint",
			method:        "GET",
			path:          "/api/health",
			statusCode:    200,
			withAuth:      false,
			expectNoEvent: true,
		},
		{
			name:          "JWKS endpoint",
			method:        "GET",
			path:          "/.well-known/jwks.json",
			statusCode:    200,
			withAuth:      false,
			expectNoEvent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			auditStore := inmemory.NewAuditStore()
			auditLogger := middleware.NewAuditLogger(auditStore)
			
			router := gin.New()
			router.Use(auditLogger.LogAuthEvents())
			
			// Add test handler
			router.Handle(tt.method, tt.path, func(c *gin.Context) {
				// Inject auth context if needed
				if tt.withAuth {
					ctx := authkit.AuthContext{
						UserID:         uuid.New(),
						Email:          "test@example.com",
						Roles:          []string{"user"},
						SessionVersion: 1,
					}
					authkit.SetContext(c, ctx)
				}
				
				c.Status(tt.statusCode)
			})
			
			// Make request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("User-Agent", "Test-Agent/1.0")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Wait for async logging
			time.Sleep(100 * time.Millisecond)
			
			// Verify
			events := auditStore.GetEvents()
			
			if tt.expectNoEvent {
				assert.Empty(t, events, "Expected no events for %s %s", tt.method, tt.path)
			} else {
				require.Len(t, events, 1, "Expected exactly one event for %s %s", tt.method, tt.path)
				
				event := events[0]
				assert.Equal(t, tt.expectedEvent, event.Type)
				
				// Check metadata
				metadata := event.Metadata
				assert.Equal(t, tt.method, metadata["method"])
				assert.Equal(t, tt.path, metadata["path"])
				assert.Equal(t, tt.statusCode, metadata["status"])
				assert.Equal(t, "Test-Agent/1.0", metadata["user_agent"])
				assert.Contains(t, metadata, "duration_ms")
				
				// Check user ID if authenticated
				if tt.withAuth {
					assert.NotNil(t, event.UserID)
					assert.NotNil(t, event.ActorID)
				}
			}
		})
	}
}

func TestAuditLogger_NoSensitiveDataLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	// Handler that would receive sensitive data
	router.POST("/auth/login/complete", func(c *gin.Context) {
		c.Status(200)
	})
	
	// Request with sensitive data
	body := `{
		"credential": {
			"clientDataJSON": "sensitive-webauthn-data",
			"authenticatorData": "more-sensitive-data",
			"signature": "cryptographic-signature"
		},
		"password": "should-never-log-this",
		"token": "jwt-token-here",
		"secret": "api-secret-key"
	}`
	
	req := httptest.NewRequest("POST", "/auth/login/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Wait for async logging
	time.Sleep(100 * time.Millisecond)
	
	// Verify no sensitive data in event
	events := auditStore.GetEvents()
	require.Len(t, events, 1)
	
	event := events[0]
	
	// Check that request body is not logged for sensitive endpoints
	assert.NotContains(t, event.Metadata, "request")
	assert.NotContains(t, event.Metadata, "body")
	assert.NotContains(t, event.Metadata, "credential")
	assert.NotContains(t, event.Metadata, "password")
	assert.NotContains(t, event.Metadata, "token")
	assert.NotContains(t, event.Metadata, "secret")
	assert.NotContains(t, event.Metadata, "clientDataJSON")
	assert.NotContains(t, event.Metadata, "authenticatorData")
	assert.NotContains(t, event.Metadata, "signature")
}

func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	
	// Handler that logs a security event
	router.GET("/test", func(c *gin.Context) {
		// Inject auth context
		ctx := authkit.AuthContext{
			UserID:         uuid.New(),
			Email:          "test@example.com",
			Roles:          []string{"admin"},
			SessionVersion: 1,
		}
		authkit.SetContext(c, ctx)
		
		// Log a security event
		metadata := map[string]interface{}{
			"reason":     "suspicious_activity",
			"ip_address": c.ClientIP(),
			"action":     "blocked",
		}
		auditLogger.LogSecurityEvent(c, string(domain.EventSuspiciousActivity), metadata)
		
		c.Status(200)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Suspicious-Bot/1.0")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Wait for async logging
	time.Sleep(100 * time.Millisecond)
	
	// Verify event
	events := auditStore.GetEvents()
	require.Len(t, events, 1)
	
	event := events[0]
	assert.Equal(t, domain.EventSuspiciousActivity, event.Type)
	assert.NotNil(t, event.UserID)
	assert.Equal(t, "suspicious_activity", event.Metadata["reason"])
	assert.Equal(t, "blocked", event.Metadata["action"])
	assert.Equal(t, "Suspicious-Bot/1.0", event.UserAgent)
}

func TestAuditLogger_ConcurrentWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	// Handler for concurrent requests
	router.POST("/auth/login/complete", func(c *gin.Context) {
		c.Status(200)
	})
	
	// Make multiple concurrent requests
	numRequests := 10
	done := make(chan bool, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest("POST", "/auth/login/complete", nil)
			req.Header.Set("User-Agent", "Test-Agent/1.0")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			done <- true
		}(i)
	}
	
	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
	
	// Wait for async logging
	time.Sleep(200 * time.Millisecond)
	
	// Verify all events were logged
	events := auditStore.GetEvents()
	assert.Len(t, events, numRequests)
	
	// All should be login success events
	for _, event := range events {
		assert.Equal(t, domain.EventLoginSuccess, event.Type)
	}
}

func TestAuditLogger_ResponseWriterWrapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	// Handler that writes data and sets status
	router.POST("/auth/login/complete", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "success"})
	})
	
	req := httptest.NewRequest("POST", "/auth/login/complete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verify response
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	
	// Wait for async logging
	time.Sleep(100 * time.Millisecond)
	
	// Verify event captured correct status
	events := auditStore.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, 200, events[0].Metadata["status"])
}

func TestAuditLogger_AsyncLoggingPerformance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	// Handler for performance testing
	router.POST("/auth/login/complete", func(c *gin.Context) {
		c.Status(200)
	})
	
	// Measure baseline response time without async logging
	req := httptest.NewRequest("POST", "/auth/login/complete", nil)
	w := httptest.NewRecorder()
	
	start := time.Now()
	router.ServeHTTP(w, req)
	responseTime := time.Since(start)
	
	// Response should be fast (async logging shouldn't block)
	assert.Less(t, responseTime, 50*time.Millisecond, "Response time should be fast with async logging")
	
	// Wait for async logging to complete
	time.Sleep(100 * time.Millisecond)
	
	// Verify event was logged
	events := auditStore.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, domain.EventLoginSuccess, events[0].Type)
}

func TestAuditLogger_AsyncLoggingUnderLoad(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	// Handler for load testing
	router.POST("/auth/login/complete", func(c *gin.Context) {
		c.Status(200)
	})
	
	// Generate high load to test async logging performance
	numConcurrentRequests := 100
	requestsPerGoroutine := 50
	totalRequests := numConcurrentRequests * requestsPerGoroutine
	
	done := make(chan time.Duration, numConcurrentRequests)
	start := time.Now()
	
	// Launch concurrent goroutines
	for i := 0; i < numConcurrentRequests; i++ {
		go func(goroutineID int) {
			goroutineStart := time.Now()
			
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("POST", "/auth/login/complete", nil)
				req.Header.Set("User-Agent", "Load-Test-Agent/1.0")
				
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				
				// Verify quick response
				if w.Code != 200 {
					t.Errorf("Unexpected status code: %d", w.Code)
				}
			}
			
			done <- time.Since(goroutineStart)
		}(i)
	}
	
	// Wait for all goroutines to complete
	var totalGoroutineTime time.Duration
	for i := 0; i < numConcurrentRequests; i++ {
		goroutineTime := <-done
		totalGoroutineTime += goroutineTime
	}
	
	totalTestTime := time.Since(start)
	avgGoroutineTime := totalGoroutineTime / time.Duration(numConcurrentRequests)
	
	t.Logf("Load test completed: %d requests in %v", totalRequests, totalTestTime)
	t.Logf("Average goroutine time: %v", avgGoroutineTime)
	t.Logf("Requests per second: %.2f", float64(totalRequests)/totalTestTime.Seconds())
	
	// Performance assertions
	assert.Less(t, totalTestTime, 10*time.Second, "Load test should complete within 10 seconds")
	assert.Less(t, avgGoroutineTime, 5*time.Second, "Average goroutine time should be reasonable")
	
	// Wait for async logging to complete
	time.Sleep(2 * time.Second)
	
	// Verify all events were logged
	events := auditStore.GetEvents()
	t.Logf("Events logged: %d out of %d expected", len(events), totalRequests)
	
	// Due to async nature and potential buffering, we allow some tolerance
	// but should get most events
	minExpectedEvents := int(float64(totalRequests) * 0.95) // 95% threshold
	assert.GreaterOrEqual(t, len(events), minExpectedEvents, 
		"Should log at least 95%% of events under load")
	
	// All logged events should be login success
	for _, event := range events {
		assert.Equal(t, domain.EventLoginSuccess, event.Type)
		assert.Contains(t, event.UserAgent, "Load-Test-Agent")
	}
}

func TestAuditLogger_AsyncLoggingMemoryUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditStore := inmemory.NewAuditStore()
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	// Handler for memory testing
	router.POST("/auth/login/complete", func(c *gin.Context) {
		// Add substantial metadata to test memory handling
		ctx := authkit.AuthContext{
			UserID:         uuid.New(),
			Email:          "test@example.com",
			Roles:          []string{"user", "admin", "developer"},
			SessionVersion: 1,
		}
		authkit.SetContext(c, ctx)
		
		c.Status(200)
	})
	
	// Create many requests to test memory usage
	numRequests := 1000
	
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("POST", "/auth/login/complete", nil)
		req.Header.Set("User-Agent", "Memory-Test-Agent/1.0")
		req.Header.Set("X-Request-ID", uuid.New().String())
		req.Header.Set("X-Correlation-ID", uuid.New().String())
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Quick verification
		assert.Equal(t, 200, w.Code)
		
		// Small delay to prevent overwhelming the system
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	
	// Wait for async processing
	time.Sleep(3 * time.Second)
	
	// Verify events were logged
	events := auditStore.GetEvents()
	t.Logf("Memory test: %d events logged out of %d requests", len(events), numRequests)
	
	// Should log most events (allowing for some async processing tolerance)
	minExpectedEvents := int(float64(numRequests) * 0.90) // 90% threshold
	assert.GreaterOrEqual(t, len(events), minExpectedEvents,
		"Should log at least 90%% of events in memory test")
	
	// Verify event structure integrity
	for i, event := range events {
		if i >= 10 { // Check first 10 events to avoid excessive logging
			break
		}
		
		assert.Equal(t, domain.EventLoginSuccess, event.Type)
		assert.NotNil(t, event.UserID, "Event %d should have UserID", i)
		assert.NotNil(t, event.ActorID, "Event %d should have ActorID", i)
		assert.Contains(t, event.Metadata, "method")
		assert.Contains(t, event.Metadata, "path")
		assert.Contains(t, event.Metadata, "status")
		assert.Contains(t, event.Metadata, "user_agent")
		assert.Contains(t, event.Metadata, "duration_ms")
	}
}

func TestAuditLogger_AsyncLoggingErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Use a mock store that can simulate failures
	auditStore := &failingAuditStore{
		events:     make([]domain.Event, 0),
		shouldFail: false,
	}
	auditLogger := middleware.NewAuditLogger(auditStore)
	
	router := gin.New()
	router.Use(auditLogger.LogAuthEvents())
	
	router.POST("/auth/login/complete", func(c *gin.Context) {
		c.Status(200)
	})
	
	// Test normal operation first
	t.Run("normal_operation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/auth/login/complete", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
		
		// Wait for async logging
		time.Sleep(100 * time.Millisecond)
		
		// Should have logged event
		events := auditStore.GetEvents()
		assert.Len(t, events, 1)
	})
	
	// Test failure handling
	t.Run("store_failure_handling", func(t *testing.T) {
		// Make store fail
		auditStore.shouldFail = true
		
		req := httptest.NewRequest("POST", "/auth/login/complete", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Response should still succeed even if audit logging fails
		assert.Equal(t, 200, w.Code)
		
		// Wait for async logging attempt
		time.Sleep(100 * time.Millisecond)
		
		// Should still only have 1 event (the previous successful one)
		events := auditStore.GetEvents()
		assert.Len(t, events, 1)
		
		// Reset for future tests
		auditStore.shouldFail = false
	})
}

// Mock audit store that can simulate failures
type failingAuditStore struct {
	events     []domain.Event
	shouldFail bool
}

func (f *failingAuditStore) Record(ctx context.Context, event domain.Event) error {
	if f.shouldFail {
		return assert.AnError
	}
	f.events = append(f.events, event)
	return nil
}

func (f *failingAuditStore) GetEvents() []domain.Event {
	return f.events
}