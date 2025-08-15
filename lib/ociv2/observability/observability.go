package observability

import (
	"fmt"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		return "unknown"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Level     LogLevel               `json:"level"`
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Operation string                 `json:"operation,omitempty"`
	Reference string                 `json:"reference,omitempty"`
	Registry  string                 `json:"registry,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger provides structured logging for OCI operations
type Logger interface {
	// Log emits a log entry at the specified level
	Log(level LogLevel, msg string, kv ...interface{})
	
	// Level-specific methods
	Debug(msg string, kv ...interface{})
	Info(msg string, kv ...interface{})
	Warn(msg string, kv ...interface{})
	Error(msg string, kv ...interface{})
	
	// Operation-specific logging
	LogOperation(operation string, ref string, registry string, duration time.Duration, err error, kv ...interface{})
	
	// WithFields returns a logger with preset fields
	WithFields(fields map[string]interface{}) Logger
}

// DefaultLogger implements Logger using the ClientOptions.Logger function
type DefaultLogger struct {
	logFunc func(msg string, kv ...interface{})
	fields  map[string]interface{}
	level   LogLevel
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(logFunc func(msg string, kv ...interface{})) Logger {
	if logFunc == nil {
		return &NoOpLogger{}
	}
	return &DefaultLogger{
		logFunc: logFunc,
		fields:  make(map[string]interface{}),
		level:   LogLevelInfo,
	}
}

// Log implements Logger.Log
func (l *DefaultLogger) Log(level LogLevel, msg string, kv ...interface{}) {
	if l.logFunc == nil || level < l.level {
		return
	}
	
	// Build key-value pairs
	pairs := []interface{}{
		"level", level.String(),
		"timestamp", time.Now().Format(time.RFC3339),
	}
	
	// Add preset fields
	for k, v := range l.fields {
		pairs = append(pairs, k, v)
	}
	
	// Add provided key-value pairs
	pairs = append(pairs, kv...)
	
	l.logFunc(msg, pairs...)
}

// Debug implements Logger.Debug
func (l *DefaultLogger) Debug(msg string, kv ...interface{}) {
	l.Log(LogLevelDebug, msg, kv...)
}

// Info implements Logger.Info
func (l *DefaultLogger) Info(msg string, kv ...interface{}) {
	l.Log(LogLevelInfo, msg, kv...)
}

// Warn implements Logger.Warn
func (l *DefaultLogger) Warn(msg string, kv ...interface{}) {
	l.Log(LogLevelWarn, msg, kv...)
}

// Error implements Logger.Error
func (l *DefaultLogger) Error(msg string, kv ...interface{}) {
	l.Log(LogLevelError, msg, kv...)
}

// LogOperation implements Logger.LogOperation
func (l *DefaultLogger) LogOperation(operation string, ref string, registry string, duration time.Duration, err error, kv ...interface{}) {
	level := LogLevelInfo
	if err != nil {
		level = LogLevelError
	}
	
	// Build operation-specific fields
	opFields := []interface{}{
		"operation", operation,
	}
	
	if ref != "" {
		opFields = append(opFields, "reference", ref)
	}
	if registry != "" {
		opFields = append(opFields, "registry", registry)
	}
	if duration > 0 {
		opFields = append(opFields, "duration_ms", duration.Milliseconds())
	}
	if err != nil {
		opFields = append(opFields, "error", err.Error())
		
		// Add error category if available
		if category := GetErrorCategory(err); category != ErrorCategoryUnknown {
			opFields = append(opFields, "error_category", string(category))
		}
		
		// Add HTTP status if available
		if status := ExtractHTTPStatus(err); status > 0 {
			opFields = append(opFields, "http_status", status)
		}
	}
	
	// Add provided fields
	opFields = append(opFields, kv...)
	
	msg := fmt.Sprintf("OCI operation: %s", operation)
	if err != nil {
		msg = fmt.Sprintf("OCI operation failed: %s", operation)
	}
	
	l.Log(level, msg, opFields...)
}

// WithFields implements Logger.WithFields
func (l *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	
	// Copy existing fields
	for k, v := range l.fields {
		newFields[k] = v
	}
	
	// Add new fields
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &DefaultLogger{
		logFunc: l.logFunc,
		fields:  newFields,
		level:   l.level,
	}
}

// NoOpLogger implements Logger but does nothing
type NoOpLogger struct{}

// Log implements Logger.Log (no-op)
func (l *NoOpLogger) Log(level LogLevel, msg string, kv ...interface{}) {}

// Debug implements Logger.Debug (no-op)
func (l *NoOpLogger) Debug(msg string, kv ...interface{}) {}

// Info implements Logger.Info (no-op)
func (l *NoOpLogger) Info(msg string, kv ...interface{}) {}

// Warn implements Logger.Warn (no-op)
func (l *NoOpLogger) Warn(msg string, kv ...interface{}) {}

// Error implements Logger.Error (no-op)
func (l *NoOpLogger) Error(msg string, kv ...interface{}) {}

// LogOperation implements Logger.LogOperation (no-op)
func (l *NoOpLogger) LogOperation(operation string, ref string, registry string, duration time.Duration, err error, kv ...interface{}) {}

// WithFields implements Logger.WithFields (returns self)
func (l *NoOpLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}

// OperationTracker tracks metrics and timing for operations
type OperationTracker struct {
	Operation string
	Reference string
	Registry  string
	StartTime time.Time
	Logger    Logger
	Fields    map[string]interface{}
}

// NewOperationTracker creates a new operation tracker
func NewOperationTracker(logger Logger, operation string, ref string, registry string) *OperationTracker {
	return &OperationTracker{
		Operation: operation,
		Reference: ref,
		Registry:  registry,
		StartTime: time.Now(),
		Logger:    logger,
		Fields:    make(map[string]interface{}),
	}
}

// WithField adds a field to the operation tracker
func (t *OperationTracker) WithField(key string, value interface{}) *OperationTracker {
	t.Fields[key] = value
	return t
}

// Start logs the beginning of an operation
func (t *OperationTracker) Start(msg string, kv ...interface{}) {
	fields := []interface{}{"operation_start", true}
	for k, v := range t.Fields {
		fields = append(fields, k, v)
	}
	fields = append(fields, kv...)
	
	t.Logger.Info(fmt.Sprintf("Starting %s: %s", t.Operation, msg), fields...)
}

// Complete logs the completion of an operation
func (t *OperationTracker) Complete(err error, kv ...interface{}) {
	duration := time.Since(t.StartTime)
	
	// Skip if no logger configured
	if t.Logger == nil {
		return
	}
	
	// Add tracker fields to the provided fields
	fields := []interface{}{}
	for k, v := range t.Fields {
		fields = append(fields, k, v)
	}
	fields = append(fields, kv...)
	
	t.Logger.LogOperation(t.Operation, t.Reference, t.Registry, duration, err, fields...)
}

// Metrics provides operation metrics and statistics
type Metrics struct {
	// Operation counts
	OperationCounts map[string]int64 `json:"operation_counts"`
	ErrorCounts     map[string]int64 `json:"error_counts"`
	
	// Registry statistics
	RegistryStats map[string]*RegistryMetrics `json:"registry_stats"`
	
	// Timing statistics
	AverageDurations map[string]time.Duration `json:"average_durations"`
	
	// Fallback statistics
	FallbackStats *FallbackMetrics `json:"fallback_stats"`
}

// RegistryMetrics tracks per-registry statistics
type RegistryMetrics struct {
	OperationCounts    map[string]int64    `json:"operation_counts"`
	ErrorCounts        map[string]int64    `json:"error_counts"`
	AverageDuration    time.Duration       `json:"average_duration"`
	ArtifactSupport    bool                `json:"artifact_support"`
	LastError          error               `json:"last_error,omitempty"`
	LastSuccessTime    time.Time           `json:"last_success_time"`
	ConsecutiveFailures int64              `json:"consecutive_failures"`
}

// FallbackMetrics tracks manifest fallback statistics
type FallbackMetrics struct {
	TotalAttempts        int64 `json:"total_attempts"`
	ArtifactSuccesses    int64 `json:"artifact_successes"`
	ImageSuccesses       int64 `json:"image_successes"`
	FallbackTriggers     int64 `json:"fallback_triggers"`
	FallbackSuccesses    int64 `json:"fallback_successes"`
	FallbackFailures     int64 `json:"fallback_failures"`
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		OperationCounts:  make(map[string]int64),
		ErrorCounts:      make(map[string]int64),
		RegistryStats:    make(map[string]*RegistryMetrics),
		AverageDurations: make(map[string]time.Duration),
		FallbackStats:    &FallbackMetrics{},
	}
}

// RecordOperation records an operation completion
func (m *Metrics) RecordOperation(operation string, registry string, duration time.Duration, err error) {
	// Update operation counts
	m.OperationCounts[operation]++
	
	// Update error counts
	if err != nil {
		category := string(GetErrorCategory(err))
		m.ErrorCounts[category]++
	}
	
	// Update registry stats
	if m.RegistryStats[registry] == nil {
		m.RegistryStats[registry] = &RegistryMetrics{
			OperationCounts: make(map[string]int64),
			ErrorCounts:     make(map[string]int64),
		}
	}
	
	regStats := m.RegistryStats[registry]
	regStats.OperationCounts[operation]++
	
	if err != nil {
		category := string(GetErrorCategory(err))
		regStats.ErrorCounts[category]++
		regStats.LastError = err
		regStats.ConsecutiveFailures++
	} else {
		regStats.LastSuccessTime = time.Now()
		regStats.ConsecutiveFailures = 0
	}
	
	// Update timing statistics (simple moving average)
	if current, exists := m.AverageDurations[operation]; exists {
		m.AverageDurations[operation] = (current + duration) / 2
	} else {
		m.AverageDurations[operation] = duration
	}
}

// RecordFallback records a manifest fallback attempt
func (m *Metrics) RecordFallback(artifactSuccess bool, imageSuccess bool) {
	m.FallbackStats.TotalAttempts++
	m.FallbackStats.FallbackTriggers++
	
	if artifactSuccess {
		m.FallbackStats.ArtifactSuccesses++
	} else if imageSuccess {
		m.FallbackStats.ImageSuccesses++
		m.FallbackStats.FallbackSuccesses++
	} else {
		m.FallbackStats.FallbackFailures++
	}
}

// GetRegistryHealth returns a health score for a registry (0-100)
func (m *Metrics) GetRegistryHealth(registry string) int {
	stats, exists := m.RegistryStats[registry]
	if !exists {
		return 100 // Unknown = assume healthy
	}
	
	totalOps := int64(0)
	totalErrors := int64(0)
	
	for _, count := range stats.OperationCounts {
		totalOps += count
	}
	for _, count := range stats.ErrorCounts {
		totalErrors += count
	}
	
	if totalOps == 0 {
		return 100
	}
	
	errorRate := float64(totalErrors) / float64(totalOps)
	health := int((1.0 - errorRate) * 100)
	
	// Penalize consecutive failures
	if stats.ConsecutiveFailures > 5 {
		health -= int(stats.ConsecutiveFailures * 2)
	}
	
	if health < 0 {
		health = 0
	}
	if health > 100 {
		health = 100
	}
	
	return health
}