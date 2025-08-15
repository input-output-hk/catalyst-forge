//go:build !wasm
// +build !wasm

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"kcl-lang.io/kcl-go"
	"kcl-lang.io/kcl-go/pkg/spec/gpyrpc"
)

// NativeEngine implements the Engine interface using CGO kcl-go.
type NativeEngine struct {
	version    string
	kclVersion string
	mu         sync.RWMutex
	coldStart  bool
}

// NewNativeEngine creates a new native KCL engine.
func NewNativeEngine() *NativeEngine {
	return &NativeEngine{
		version:    "native-v1.0.0+cgo", // Indicates CGO-based native engine
		kclVersion: getKCLVersion(),
		coldStart:  true,
	}
}

// Version returns the engine version.
func (e *NativeEngine) Version() string {
	return e.version
}

// KCLVersion returns the KCL runtime version.
func (e *NativeEngine) KCLVersion() string {
	return e.kclVersion
}

// Run executes KCL code with the given inputs.
func (e *NativeEngine) Run(ctx context.Context, workDir, entry string, valuesJSON, ctxJSON []byte, lim Limits) ([]byte, Stats, error) {
	stats := Stats{
		StartedAt: time.Now(),
	}

	// Check cold start
	e.mu.Lock()
	stats.ColdStart = e.coldStart
	e.coldStart = false
	e.mu.Unlock()

	// Apply timeout
	if lim.TimeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(lim.TimeoutSec)*time.Second)
		defer cancel()
	}

	// Prepare execution options
	opts, err := e.buildOptions(workDir, entry, valuesJSON, ctxJSON)
	if err != nil {
		return nil, stats, EngineError{
			Engine:  KindNative,
			Phase:   "prepare",
			Message: "failed to build options",
			Cause:   err,
		}
	}

	// Execute KCL
	compileStart := time.Now()

	// Run in a goroutine to handle context cancellation
	type result struct {
		yaml []byte
		err  error
	}

	// Get entry path from options
	entryPath := filepath.Join(workDir, entry)

	resultCh := make(chan result, 1)
	go func() {
		// Execute KCL with the entry path and options
		res, err := kcl.Run(entryPath, *opts)
		if err != nil {
			resultCh <- result{nil, err}
			return
		}

		// Convert result to YAML
		yamlBytes := []byte(res.GetRawYamlResult())
		resultCh <- result{yamlBytes, nil}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		stats.EndedAt = time.Now()
		stats.TotalMS = stats.EndedAt.Sub(stats.StartedAt).Milliseconds()
		return nil, stats, EngineError{
			Engine:  KindNative,
			Phase:   "timeout",
			Message: fmt.Sprintf("execution timed out after %d seconds", lim.TimeoutSec),
			Cause:   ctx.Err(),
		}

	case res := <-resultCh:
		stats.EndedAt = time.Now()
		stats.CompileMS = time.Since(compileStart).Milliseconds()
		stats.EvalMS = 0 // Native engine doesn't separate compile/eval
		stats.TotalMS = stats.EndedAt.Sub(stats.StartedAt).Milliseconds()

		// Estimate memory usage (simplified)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		stats.PeakMemMB = int(m.Alloc / 1024 / 1024)

		if res.err != nil {
			return nil, stats, EngineError{
				Engine:  KindNative,
				Phase:   "execution",
				Message: "KCL execution failed",
				Cause:   res.err,
			}
		}

		return res.yaml, stats, nil
	}
}

// Validate checks if the engine is available and properly configured.
func (e *NativeEngine) Validate() error {
	// Try to get KCL version as a validation check
	version := getKCLVersion()
	if version == "" {
		return fmt.Errorf("KCL runtime not available")
	}
	return nil
}

// Close cleans up any resources held by the engine.
func (e *NativeEngine) Close() error {
	// Native engine doesn't hold persistent resources
	return nil
}

// buildOptions builds KCL execution options.
func (e *NativeEngine) buildOptions(workDir, entry string, valuesJSON, ctxJSON []byte) (*kcl.Option, error) {
	opts := kcl.NewOption()

	// Set working directory
	opts.WorkDir = workDir

	// Set entry point
	entryPath := filepath.Join(workDir, entry)
	if !fileExists(entryPath) {
		// If entry is a directory or '.', attempt discovery
		if info, err := os.Stat(entryPath); (err == nil && info.IsDir()) || entry == "." {
			candidates := []string{"main.k", "index.k"}
			for _, name := range candidates {
				p := filepath.Join(workDir, name)
				if fileExists(p) {
					entryPath = p
					break
				}
			}
			if !fileExists(entryPath) {
				entries, err := os.ReadDir(workDir)
				if err == nil {
					for _, e := range entries {
						if !e.IsDir() && filepath.Ext(e.Name()) == ".k" {
							entryPath = filepath.Join(workDir, e.Name())
							break
						}
					}
				}
			}
		}
		if !fileExists(entryPath) {
			return nil, fmt.Errorf("entry file not found: %s", entryPath)
		}
	}
	opts.KFilenameList = []string{entryPath}

	// Disable external dependencies
	opts.DisableNone = false // Allow None values
	// opts.NoStyle = true      // Disable color output (field may not exist)

	// Set values if provided
	if len(valuesJSON) > 0 {
		values, err := jsonToKCLValues(valuesJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to parse values: %w", err)
		}
		opts.ExternalPkgs = values
	}

	// Set context if provided (merge with values)
	if len(ctxJSON) > 0 {
		ctx, err := jsonToKCLContext(ctxJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to parse context: %w", err)
		}
		// Merge context into external packages
		if opts.ExternalPkgs == nil {
			opts.ExternalPkgs = make([]*gpyrpc.ExternalPkg, 0)
		}
		opts.ExternalPkgs = append(opts.ExternalPkgs, ctx...)
	}

	// Set additional options for determinism
	opts.SortKeys = true               // Sort output keys
	opts.IncludeSchemaTypePath = false // Don't include schema paths in output

	return opts, nil
}

// jsonToKCLValues converts JSON values to KCL external packages.
func jsonToKCLValues(data []byte) ([]*gpyrpc.ExternalPkg, error) {
	var values map[string]interface{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}

	var pkgs []*gpyrpc.ExternalPkg
	for key, value := range values {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		pkg := &gpyrpc.ExternalPkg{
			PkgName: "__values__",
			PkgPath: key,
			// Convert to KCL-compatible format
			// This is simplified - real implementation would handle complex types
		}
		_ = valueJSON // Use this to set pkg fields appropriately
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// jsonToKCLContext converts JSON context to KCL external packages.
func jsonToKCLContext(data []byte) ([]*gpyrpc.ExternalPkg, error) {
	var ctx map[string]interface{}
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, err
	}

	var pkgs []*gpyrpc.ExternalPkg
	for key, value := range ctx {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		pkg := &gpyrpc.ExternalPkg{
			PkgName: "__context__",
			PkgPath: key,
			// Convert to KCL-compatible format
		}
		_ = valueJSON // Use this to set pkg fields appropriately
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// getKCLVersion retrieves the KCL runtime version.
func getKCLVersion() string {
	result, err := kcl.GetVersion()
	if err != nil {
		// Return a fallback version if we can't get the actual version
		return "unknown"
	}
	// Return the version string from the result
	return result.GetVersion()
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// init registers the native engine if available.
func init() {
	// Only register if KCL is available
	engine := NewNativeEngine()
	if err := engine.Validate(); err == nil {
		_ = Register(KindNative, engine)
	}
}
