//go:build wasm
// +build wasm

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wasmerio/wasmer-go/wasmer"
)

// WASMEngine implements the Engine interface using WASM/WASI runtime.
type WASMEngine struct {
	version    string
	kclVersion string
	wasmPath   string
	mu         sync.RWMutex
	coldStart  bool
	store      *wasmer.Store
	module     *wasmer.Module
	instance   *wasmer.Instance
	initOnce   sync.Once
}

// NewWASMEngine creates a new WASM KCL engine.
func NewWASMEngine(wasmPath string) *WASMEngine {
	if wasmPath == "" {
		wasmPath = os.Getenv("KCL_WASM_PATH")
		if wasmPath == "" {
			wasmPath = "/usr/local/lib/kcl/kcl.wasm"
		}
	}
	
	return &WASMEngine{
		version:    "wasm-v1.0.0",
		kclVersion: "0.9.0-wasm", // This would be extracted from WASM module
		wasmPath:   wasmPath,
		coldStart:  true,
	}
}

// Version returns the engine version.
func (e *WASMEngine) Version() string {
	return e.version
}

// KCLVersion returns the KCL runtime version.
func (e *WASMEngine) KCLVersion() string {
	return e.kclVersion
}

// Run executes KCL code with the given inputs.
func (e *WASMEngine) Run(ctx context.Context, workDir, entry string, valuesJSON, ctxJSON []byte, lim Limits) ([]byte, Stats, error) {
	stats := Stats{
		StartedAt: time.Now(),
	}

	// Initialize WASM module if needed
	e.initOnce.Do(func() {
		if err := e.initialize(); err != nil {
			// Store error for later
			e.mu.Lock()
			e.coldStart = false
			e.mu.Unlock()
		}
	})

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

	// Create WASI environment
	wasiEnv, err := e.createWASIEnv(workDir, lim.MemoryLimitMB)
	if err != nil {
		return nil, stats, EngineError{
			Engine:  KindWASM,
			Phase:   "prepare",
			Message: "failed to create WASI environment",
			Cause:   err,
		}
	}

	// Prepare input
	input, err := e.prepareInput(workDir, entry, valuesJSON, ctxJSON)
	if err != nil {
		return nil, stats, EngineError{
			Engine:  KindWASM,
			Phase:   "prepare",
			Message: "failed to prepare input",
			Cause:   err,
		}
	}

	// Execute in goroutine for timeout handling
	type result struct {
		output []byte
		err    error
	}
	
	resultCh := make(chan result, 1)
	compileStart := time.Now()
	
	go func() {
		output, err := e.execute(wasiEnv, input)
		resultCh <- result{output, err}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		stats.EndedAt = time.Now()
		stats.TotalMS = stats.EndedAt.Sub(stats.StartedAt).Milliseconds()
		return nil, stats, EngineError{
			Engine:  KindWASM,
			Phase:   "timeout",
			Message: fmt.Sprintf("execution timed out after %d seconds", lim.TimeoutSec),
			Cause:   ctx.Err(),
		}
		
	case res := <-resultCh:
		stats.EndedAt = time.Now()
		stats.CompileMS = time.Since(compileStart).Milliseconds() / 2 // Estimate
		stats.EvalMS = stats.CompileMS                                // Estimate
		stats.TotalMS = stats.EndedAt.Sub(stats.StartedAt).Milliseconds()
		
		// Get memory stats from WASI environment
		stats.PeakMemMB = e.getMemoryUsage(wasiEnv)
		
		if res.err != nil {
			return nil, stats, EngineError{
				Engine:  KindWASM,
				Phase:   "execution",
				Message: "WASM execution failed",
				Cause:   res.err,
			}
		}
		
		return res.output, stats, nil
	}
}

// Validate checks if the engine is available and properly configured.
func (e *WASMEngine) Validate() error {
	// Check if WASM file exists
	if _, err := os.Stat(e.wasmPath); err != nil {
		return fmt.Errorf("KCL WASM module not found at %s: %w", e.wasmPath, err)
	}
	return nil
}

// Close cleans up any resources held by the engine.
func (e *WASMEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.instance != nil {
		// Clean up WASM instance
		e.instance = nil
	}
	if e.module != nil {
		// Clean up WASM module
		e.module = nil
	}
	if e.store != nil {
		// Clean up WASM store
		e.store = nil
	}
	
	return nil
}

// initialize loads and prepares the WASM module.
func (e *WASMEngine) initialize() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Read WASM bytes
	wasmBytes, err := os.ReadFile(e.wasmPath)
	if err != nil {
		return fmt.Errorf("failed to read WASM module: %w", err)
	}

	// Create store
	engine := wasmer.NewEngine()
	e.store = wasmer.NewStore(engine)

	// Compile module
	e.module, err = wasmer.NewModule(e.store, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile WASM module: %w", err)
	}

	return nil
}

// createWASIEnv creates a WASI environment for execution.
func (e *WASMEngine) createWASIEnv(workDir string, memLimitMB int) (*wasmer.WasiEnvironment, error) {
	// Create WASI config
	wasiConfig := wasmer.NewWasiStateBuilder("kcl").
		PreopenDirectory(workDir).
		MapDirectory("/work", workDir).
		CaptureStdout().
		CaptureStderr()

	// Apply memory limit if specified
	if memLimitMB > 0 {
		// This would set memory limits on the WASI environment
		// Implementation depends on wasmer-go capabilities
	}

	wasiEnv, err := wasiConfig.Finalize()
	if err != nil {
		return nil, err
	}

	return wasiEnv, nil
}

// prepareInput prepares input for WASM execution.
func (e *WASMEngine) prepareInput(workDir, entry string, valuesJSON, ctxJSON []byte) ([]byte, error) {
	// Build input structure for WASM module
	input := map[string]interface{}{
		"workDir": workDir,
		"entry":   entry,
	}

	if len(valuesJSON) > 0 {
		var values interface{}
		if err := json.Unmarshal(valuesJSON, &values); err != nil {
			return nil, err
		}
		input["values"] = values
	}

	if len(ctxJSON) > 0 {
		var ctx interface{}
		if err := json.Unmarshal(ctxJSON, &ctx); err != nil {
			return nil, err
		}
		input["context"] = ctx
	}

	return json.Marshal(input)
}

// execute runs the WASM module with input.
func (e *WASMEngine) execute(wasiEnv *wasmer.WasiEnvironment, input []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.module == nil {
		return nil, fmt.Errorf("WASM module not initialized")
	}

	// Create import object
	importObject := wasiEnv.GenerateImportObject(e.store, e.module)

	// Instantiate module
	instance, err := wasmer.NewInstance(e.module, importObject)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	// Get memory
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return nil, fmt.Errorf("failed to get WASM memory: %w", err)
	}

	// Allocate memory for input
	allocFn, err := instance.Exports.GetFunction("allocate")
	if err != nil {
		return nil, fmt.Errorf("failed to get allocate function: %w", err)
	}

	inputLen := len(input)
	allocResult, err := allocFn(inputLen)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory: %w", err)
	}

	inputPtr := allocResult.(int32)

	// Write input to memory
	memory.Data()[inputPtr:inputPtr+int32(inputLen)] = input

	// Call main function
	mainFn, err := instance.Exports.GetFunction("kcl_run")
	if err != nil {
		return nil, fmt.Errorf("failed to get kcl_run function: %w", err)
	}

	result, err := mainFn(inputPtr, inputLen)
	if err != nil {
		return nil, fmt.Errorf("KCL execution failed: %w", err)
	}

	// Read output
	outputPtr := result.(int32)
	
	// Get output length
	getLenFn, err := instance.Exports.GetFunction("get_output_len")
	if err != nil {
		return nil, fmt.Errorf("failed to get output length function: %w", err)
	}

	lenResult, err := getLenFn()
	if err != nil {
		return nil, fmt.Errorf("failed to get output length: %w", err)
	}

	outputLen := lenResult.(int32)

	// Read output from memory
	output := make([]byte, outputLen)
	copy(output, memory.Data()[outputPtr:outputPtr+outputLen])

	// Free memory
	freeFn, err := instance.Exports.GetFunction("free")
	if err == nil {
		_, _ = freeFn(inputPtr, inputLen)
		_, _ = freeFn(outputPtr, outputLen)
	}

	return output, nil
}

// getMemoryUsage returns memory usage in MB.
func (e *WASMEngine) getMemoryUsage(wasiEnv *wasmer.WasiEnvironment) int {
	// This would query WASI environment for memory usage
	// Implementation depends on wasmer-go capabilities
	return 0
}

// init registers the WASM engine if available.
func init() {
	// Only register if WASM runtime is available
	engine := NewWASMEngine("")
	if err := engine.Validate(); err == nil {
		_ = Register(KindWASM, engine)
	}
}

// Stdout captures stdout from WASI.
type Stdout struct {
	data []byte
	mu   sync.Mutex
}

func (s *Stdout) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data, p...)
	return len(p), nil
}

func (s *Stdout) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data) == 0 {
		return 0, io.EOF
	}
	n = copy(p, s.data)
	s.data = s.data[n:]
	return n, nil
}

func (s *Stdout) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.data)
}