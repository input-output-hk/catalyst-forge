package engine

import (
	"context"
	"fmt"
	"time"
)

// Engine defines the interface for KCL execution backends.
type Engine interface {
	// Version returns the engine build/version (affects intent hash).
	Version() string

	// KCLVersion returns the reported KCL runtime version (affects intent hash).
	KCLVersion() string

	// Run executes KCL code with the given inputs.
	Run(ctx context.Context, workDir, entry string, valuesJSON, ctxJSON []byte, lim Limits) (out []byte, stats Stats, err error)

	// Validate checks if the engine is available and properly configured.
	Validate() error

	// Close cleans up any resources held by the engine.
	Close() error
}

// Limits defines resource limits for KCL execution.
type Limits struct {
	TimeoutSec    int // Execution timeout in seconds
	MemoryLimitMB int // Memory limit in MB (primarily for WASM)
}

// Stats contains execution statistics.
type Stats struct {
	ColdStart bool          // True if engine was just initialized
	CompileMS int64         // Compilation time in milliseconds
	EvalMS    int64         // Evaluation time in milliseconds
	TotalMS   int64         // Total execution time in milliseconds
	PeakMemMB int           // Peak memory usage in MB
	StartedAt time.Time     // When execution started
	EndedAt   time.Time     // When execution ended
}

// Kind defines the type of execution engine.
type Kind string

const (
	// KindNative uses CGO kcl-go for high performance.
	KindNative Kind = "native"
	// KindWASM uses WASM/WASI runtime for isolation.
	KindWASM Kind = "wasm"
)

// Registry holds available engines.
type Registry struct {
	engines map[Kind]Engine
}

// NewRegistry creates a new engine registry.
func NewRegistry() *Registry {
	return &Registry{
		engines: make(map[Kind]Engine),
	}
}

// Register adds an engine to the registry.
func (r *Registry) Register(kind Kind, engine Engine) error {
	if engine == nil {
		return fmt.Errorf("engine cannot be nil")
	}
	
	if err := engine.Validate(); err != nil {
		return fmt.Errorf("engine validation failed: %w", err)
	}
	
	r.engines[kind] = engine
	return nil
}

// Get retrieves an engine from the registry.
func (r *Registry) Get(kind Kind) (Engine, error) {
	engine, exists := r.engines[kind]
	if !exists {
		return nil, fmt.Errorf("engine %s not registered", kind)
	}
	return engine, nil
}

// Has checks if an engine is registered.
func (r *Registry) Has(kind Kind) bool {
	_, exists := r.engines[kind]
	return exists
}

// Close closes all registered engines.
func (r *Registry) Close() error {
	var firstErr error
	for _, engine := range r.engines {
		if err := engine.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// DefaultRegistry is the global engine registry.
var DefaultRegistry = NewRegistry()

// Register registers an engine in the default registry.
func Register(kind Kind, engine Engine) error {
	return DefaultRegistry.Register(kind, engine)
}

// Get retrieves an engine from the default registry.
func Get(kind Kind) (Engine, error) {
	return DefaultRegistry.Get(kind)
}

// SelectEngine selects the best available engine.
func SelectEngine(preferred Kind) (Engine, error) {
	// Try preferred engine first
	if engine, err := Get(preferred); err == nil {
		return engine, nil
	}
	
	// Fallback order
	fallbacks := []Kind{KindNative, KindWASM}
	for _, kind := range fallbacks {
		if engine, err := Get(kind); err == nil {
			return engine, nil
		}
	}
	
	return nil, fmt.Errorf("no engine available")
}

// EngineError represents an engine-specific error.
type EngineError struct {
	Engine  Kind
	Phase   string // "compile", "eval", "timeout", etc.
	Message string
	Cause   error
}

func (e EngineError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s engine %s error: %s: %v", e.Engine, e.Phase, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s engine %s error: %s", e.Engine, e.Phase, e.Message)
}

func (e EngineError) Unwrap() error {
	return e.Cause
}

// Helper functions for working with engines

// PrepareWorkDir prepares a working directory for KCL execution.
func PrepareWorkDir(workDir, entry string) error {
	// This would validate that:
	// 1. workDir exists and is a directory
	// 2. entry file exists within workDir
	// 3. Permissions are correct
	// Implementation depends on requirements
	return nil
}

// MergeValues merges multiple value sources for KCL execution.
func MergeValues(sources ...[]byte) ([]byte, error) {
	// This would merge multiple JSON value sources
	// Implementation depends on merge strategy
	return nil, nil
}

// ValidateOutput validates KCL output is valid YAML/JSON.
func ValidateOutput(output []byte) error {
	// Basic validation that output is valid YAML
	// Implementation depends on validation requirements
	return nil
}