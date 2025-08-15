package ociv2

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevel_String(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "debug"},
		{LogLevelInfo, "info"},
		{LogLevelWarn, "warn"},
		{LogLevelError, "error"},
		{LogLevel(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	t.Parallel()
	
	t.Run("WithNilLogFunc", func(t *testing.T) {
		logger := NewDefaultLogger(nil)
		assert.IsType(t, &NoOpLogger{}, logger)
	})
	
	t.Run("WithLogFunc", func(t *testing.T) {
		var capturedMsg string
		var capturedKV []interface{}
		
		logFunc := func(msg string, kv ...interface{}) {
			capturedMsg = msg
			capturedKV = kv
		}
		
		logger := NewDefaultLogger(logFunc)
		
		// Test basic logging
		logger.Info("test message", "key", "value")
		
		assert.Equal(t, "test message", capturedMsg)
		assert.Contains(t, capturedKV, "level")
		assert.Contains(t, capturedKV, "info")
		assert.Contains(t, capturedKV, "key")
		assert.Contains(t, capturedKV, "value")
	})
	
	t.Run("WithFields", func(t *testing.T) {
		var capturedKV []interface{}
		
		logFunc := func(msg string, kv ...interface{}) {
			capturedKV = kv
		}
		
		logger := NewDefaultLogger(logFunc)
		fieldsLogger := logger.WithFields(map[string]interface{}{
			"operation": "test",
			"registry":  "example.com",
		})
		
		fieldsLogger.Info("test message")
		
		assert.Contains(t, capturedKV, "operation")
		assert.Contains(t, capturedKV, "test")
		assert.Contains(t, capturedKV, "registry")
		assert.Contains(t, capturedKV, "example.com")
	})
	
	t.Run("LogOperation", func(t *testing.T) {
		var capturedMsg string
		var capturedKV []interface{}
		
		logFunc := func(msg string, kv ...interface{}) {
			capturedMsg = msg
			capturedKV = kv
		}
		
		logger := NewDefaultLogger(logFunc)
		
		// Test successful operation
		logger.LogOperation("push", "example.com/repo:tag", "example.com", 100*time.Millisecond, nil)
		
		assert.Equal(t, "OCI operation: push", capturedMsg)
		assert.Contains(t, capturedKV, "operation")
		assert.Contains(t, capturedKV, "push")
		assert.Contains(t, capturedKV, "reference")
		assert.Contains(t, capturedKV, "example.com/repo:tag")
		assert.Contains(t, capturedKV, "registry")
		assert.Contains(t, capturedKV, "example.com")
		assert.Contains(t, capturedKV, "duration_ms")
		assert.Contains(t, capturedKV, int64(100))
		
		// Test failed operation
		testErr := errors.New("test error")
		logger.LogOperation("pull", "example.com/repo:tag", "example.com", 200*time.Millisecond, testErr)
		
		assert.Equal(t, "OCI operation failed: pull", capturedMsg)
		assert.Contains(t, capturedKV, "error")
		assert.Contains(t, capturedKV, "test error")
	})
}

func TestNoOpLogger(t *testing.T) {
	t.Parallel()
	
	logger := &NoOpLogger{}
	
	// These should not panic
	logger.Log(LogLevelInfo, "test")
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.LogOperation("test", "ref", "registry", time.Second, nil)
	
	// WithFields should return self
	fieldsLogger := logger.WithFields(map[string]interface{}{"key": "value"})
	assert.Equal(t, logger, fieldsLogger)
}

func TestOperationTracker(t *testing.T) {
	t.Parallel()
	
	t.Run("BasicTracking", func(t *testing.T) {
		var capturedEntries []string
		var capturedKV [][]interface{}
		
		logFunc := func(msg string, kv ...interface{}) {
			capturedEntries = append(capturedEntries, msg)
			capturedKV = append(capturedKV, kv)
		}
		
		logger := NewDefaultLogger(logFunc)
		tracker := NewOperationTracker(logger, "test_op", "example.com/repo:tag", "example.com")
		
		// Add some fields
		tracker.WithField("custom", "value")
		
		// Start tracking
		tracker.Start("starting test operation")
		
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		
		// Complete tracking
		tracker.Complete(nil, "extra", "data")
		
		// Verify start log
		assert.Contains(t, capturedEntries[0], "Starting test_op: starting test operation")
		assert.Contains(t, capturedKV[0], "operation_start")
		assert.Contains(t, capturedKV[0], true)
		assert.Contains(t, capturedKV[0], "custom")
		assert.Contains(t, capturedKV[0], "value")
		
		// Verify completion log
		assert.Equal(t, "OCI operation: test_op", capturedEntries[1])
		assert.Contains(t, capturedKV[1], "operation")
		assert.Contains(t, capturedKV[1], "test_op")
		assert.Contains(t, capturedKV[1], "reference")
		assert.Contains(t, capturedKV[1], "example.com/repo:tag")
		assert.Contains(t, capturedKV[1], "registry")
		assert.Contains(t, capturedKV[1], "example.com")
		assert.Contains(t, capturedKV[1], "duration_ms")
		assert.Contains(t, capturedKV[1], "custom")
		assert.Contains(t, capturedKV[1], "value")
		assert.Contains(t, capturedKV[1], "extra")
		assert.Contains(t, capturedKV[1], "data")
	})
	
	t.Run("TrackingWithError", func(t *testing.T) {
		var capturedEntries []string
		
		logFunc := func(msg string, kv ...interface{}) {
			capturedEntries = append(capturedEntries, msg)
		}
		
		logger := NewDefaultLogger(logFunc)
		tracker := NewOperationTracker(logger, "test_op", "example.com/repo:tag", "example.com")
		
		tracker.Start("starting test operation")
		
		testErr := errors.New("test error")
		tracker.Complete(testErr)
		
		// Should log as failed operation
		assert.Equal(t, "OCI operation failed: test_op", capturedEntries[1])
	})
}

func TestMetrics(t *testing.T) {
	t.Parallel()
	
	t.Run("NewMetrics", func(t *testing.T) {
		metrics := NewMetrics()
		
		assert.NotNil(t, metrics.OperationCounts)
		assert.NotNil(t, metrics.ErrorCounts)
		assert.NotNil(t, metrics.RegistryStats)
		assert.NotNil(t, metrics.AverageDurations)
		assert.NotNil(t, metrics.FallbackStats)
		assert.Equal(t, int64(0), metrics.FallbackStats.TotalAttempts)
	})
	
	t.Run("RecordOperation", func(t *testing.T) {
		metrics := NewMetrics()
		
		// Record successful operation
		metrics.RecordOperation("push", "example.com", 100*time.Millisecond, nil)
		
		assert.Equal(t, int64(1), metrics.OperationCounts["push"])
		assert.Equal(t, 100*time.Millisecond, metrics.AverageDurations["push"])
		
		// Verify registry stats
		regStats := metrics.RegistryStats["example.com"]
		require.NotNil(t, regStats)
		assert.Equal(t, int64(1), regStats.OperationCounts["push"])
		assert.Equal(t, int64(0), regStats.ConsecutiveFailures)
		assert.False(t, regStats.LastSuccessTime.IsZero())
		
		// Record failed operation
		testErr := NewNetworkError("pull", "example.com", errors.New("connection failed"))
		metrics.RecordOperation("pull", "example.com", 200*time.Millisecond, testErr)
		
		assert.Equal(t, int64(1), metrics.OperationCounts["pull"])
		assert.Equal(t, int64(1), metrics.ErrorCounts["network"])
		
		// Registry stats should show failure
		assert.Equal(t, int64(1), regStats.OperationCounts["pull"])
		assert.Equal(t, int64(1), regStats.ErrorCounts["network"])
		assert.Equal(t, int64(1), regStats.ConsecutiveFailures)
		assert.Equal(t, testErr, regStats.LastError)
	})
	
	t.Run("RecordFallback", func(t *testing.T) {
		metrics := NewMetrics()
		
		// Record artifact success
		metrics.RecordFallback(true, false)
		
		assert.Equal(t, int64(1), metrics.FallbackStats.TotalAttempts)
		assert.Equal(t, int64(1), metrics.FallbackStats.FallbackTriggers)
		assert.Equal(t, int64(1), metrics.FallbackStats.ArtifactSuccesses)
		assert.Equal(t, int64(0), metrics.FallbackStats.ImageSuccesses)
		
		// Record image fallback success
		metrics.RecordFallback(false, true)
		
		assert.Equal(t, int64(2), metrics.FallbackStats.TotalAttempts)
		assert.Equal(t, int64(2), metrics.FallbackStats.FallbackTriggers)
		assert.Equal(t, int64(1), metrics.FallbackStats.ImageSuccesses)
		assert.Equal(t, int64(1), metrics.FallbackStats.FallbackSuccesses)
		
		// Record complete failure
		metrics.RecordFallback(false, false)
		
		assert.Equal(t, int64(3), metrics.FallbackStats.TotalAttempts)
		assert.Equal(t, int64(1), metrics.FallbackStats.FallbackFailures)
	})
	
	t.Run("GetRegistryHealth", func(t *testing.T) {
		metrics := NewMetrics()
		
		// Unknown registry should be healthy
		health := metrics.GetRegistryHealth("unknown.com")
		assert.Equal(t, 100, health)
		
		// Record some successful operations
		for i := 0; i < 10; i++ {
			metrics.RecordOperation("push", "example.com", 100*time.Millisecond, nil)
		}
		
		health = metrics.GetRegistryHealth("example.com")
		assert.Equal(t, 100, health)
		
		// Add some failures
		testErr := errors.New("test error")
		for i := 0; i < 2; i++ {
			metrics.RecordOperation("push", "example.com", 100*time.Millisecond, testErr)
		}
		
		// Health should be 80% (10 success, 2 failures = 10/12 = 83%, rounded to 83)
		health = metrics.GetRegistryHealth("example.com")
		assert.True(t, health >= 80 && health <= 85)
		
		// Add consecutive failures to penalize health
		regStats := metrics.RegistryStats["example.com"]
		regStats.ConsecutiveFailures = 10
		
		health = metrics.GetRegistryHealth("example.com")
		assert.True(t, health < 80) // Should be penalized
	})
}

func TestErrorIntegration(t *testing.T) {
	t.Parallel()
	
	t.Run("LoggerWithStructuredErrors", func(t *testing.T) {
		var capturedKV []interface{}
		
		logFunc := func(msg string, kv ...interface{}) {
			capturedKV = kv
		}
		
		logger := NewDefaultLogger(logFunc)
		
		// Create a structured error
		ociErr := NewAuthError("push", "example.com", errors.New("invalid token"))
		
		// Log operation with structured error
		logger.LogOperation("push", "example.com/repo:tag", "example.com", 100*time.Millisecond, ociErr)
		
		// Should include error category and HTTP status if available
		assert.Contains(t, capturedKV, "error_category")
		assert.Contains(t, capturedKV, "auth")
	})
	
	t.Run("MetricsWithStructuredErrors", func(t *testing.T) {
		metrics := NewMetrics()
		
		// Record operation with structured error
		ociErr := NewRegistryError("push", "example.com", 404, errors.New("not found"))
		metrics.RecordOperation("push", "example.com", 100*time.Millisecond, ociErr)
		
		// Should categorize error correctly
		assert.Equal(t, int64(1), metrics.ErrorCounts["registry"])
		
		regStats := metrics.RegistryStats["example.com"]
		assert.Equal(t, int64(1), regStats.ErrorCounts["registry"])
	})
}

// Test integration between logger and metrics with a mock client operation
func TestObservabilityIntegration(t *testing.T) {
	t.Parallel()
	
	var logEntries []string
	var logKV [][]interface{}
	var metricsCallbacks []*Metrics
	
	logFunc := func(msg string, kv ...interface{}) {
		logEntries = append(logEntries, msg)
		logKV = append(logKV, kv)
	}
	
	metricsFunc := func(m *Metrics) {
		metricsCallbacks = append(metricsCallbacks, m)
	}
	
	// Create client with observability enabled
	opts := ClientOptions{
		StructuredLogger: NewDefaultLogger(logFunc),
		EnableMetrics:    true,
		MetricsCallback:  metricsFunc,
	}
	
	ociClient, err := New(opts)
	require.NoError(t, err)
	
	// Test that the client has the observability options set
	impl, ok := ociClient.(*client)
	require.True(t, ok)
	assert.NotNil(t, impl.opts.StructuredLogger)
	assert.True(t, impl.opts.EnableMetrics)
	assert.NotNil(t, impl.opts.MetricsCallback)
}