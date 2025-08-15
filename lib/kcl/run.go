package kcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/cache"
	"github.com/input-output-hk/catalyst-forge/lib/kcl/engine"
	"github.com/input-output-hk/catalyst-forge/lib/kcl/internal"
)

// Run executes a KCL module with the given inputs, using caching to avoid repeated evaluation.
func Run(ctx context.Context, oci OCI, ref ModuleRef, opts RunOptions) (*RunResult, error) {
	// Initialize cache manager
	cm, err := cache.GetManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache manager: %w", err)
	}

	// Step 1: Pull module to ensure it's in cache
	pullOpts := PullOptions{
		Profile:          opts.Profile,
		RequireSignature: opts.RequireSignature,
		SkipVerify:       opts.SkipVerify,
		Progress:         opts.Progress,
	}

	pullResult, err := Pull(ctx, oci, ref, pullOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to pull module: %w", err)
	}

	// Step 2: Select engine
	selectedEngine, err := selectEngine(opts.Engine)
	if err != nil {
		return nil, fmt.Errorf("failed to select engine: %w", err)
	}

	// Step 3: Compute intent hash
	intentHash, err := computeIntentHash(
		opts.Profile,
		pullResult.Digest,
		opts.Engine,
		selectedEngine,
		opts.Values,
		opts.Context,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compute intent hash: %w", err)
	}

	// Step 4: Check run cache if enabled
	if opts.UseCache && !opts.ForceRecompute {
		cached, err := loadCachedRun(cm, intentHash, pullResult.Digest, pullResult.Meta)
		if err == nil && cached != nil {
			// Update access time
			_ = touchRunCache(cm, intentHash)
			return cached, nil
		}
		// Cache miss or error, continue to execution
	}

	// Step 5: Execute with lock and singleflight
	var runResult *RunResult

	// Use singleflight to dedupe in-process
	result, err := cm.SingleflightRun(intentHash, func() (interface{}, error) {
		// Acquire cross-process lock
		return executeWithLock(ctx, cm, intentHash, func() (*RunResult, error) {
			// Check cache again inside lock (another process may have completed)
			if opts.UseCache && !opts.ForceRecompute {
				cached, err := loadCachedRun(cm, intentHash, pullResult.Digest, pullResult.Meta)
				if err == nil && cached != nil {
					return cached, nil
				}
			}

			// Execute engine
			return executeEngine(ctx, selectedEngine, pullResult, opts, intentHash, cm)
		})
	})

	if err != nil {
		return nil, err
	}
	runResult = result.(*RunResult)

	return runResult, nil
}

// executeWithLock executes a function while holding a cross-process lock
func executeWithLock(ctx context.Context, cm *cache.Manager, intentHash string, fn func() (*RunResult, error)) (*RunResult, error) {
	recovered := (*RunResult)(nil)
	err := cm.WithRunLock(intentHash, func() error {
		result, err := fn()
		if err != nil {
			return err
		}
		// Store result for return outside lock without using context
		recovered = result
		return nil
	})

	if err != nil {
		return nil, err
	}

	if recovered == nil {
		return nil, fmt.Errorf("failed to retrieve run result")
	}
	return recovered, nil
}

// executeEngine runs the KCL engine and caches the result
func executeEngine(ctx context.Context, eng engine.Engine, pullResult *PullResult, opts RunOptions, intentHash string, cm *cache.Manager) (*RunResult, error) {
	// Derive entry point
	entry := deriveEntry(pullResult.Meta, opts.Profile, pullResult.Path)

	// Prepare limits
	limits := engine.Limits{
		TimeoutSec:    opts.TimeoutSec,
		MemoryLimitMB: opts.MemoryLimitMB,
	}

	// Execute engine
	output, stats, err := eng.Run(ctx, pullResult.Path, entry, opts.Values, opts.Context, limits)
	if err != nil {
		return nil, fmt.Errorf("engine execution failed: %w", err)
	}

	// Create result
	result := &RunResult{
		YAML:     output,
		Digest:   pullResult.Digest,
		MetaJSON: pullResult.MetaJSON,
		Stats: EngineStats{
			ColdStart: stats.ColdStart,
			CompileMS: stats.CompileMS,
			EvalMS:    stats.EvalMS,
			PeakMemMB: stats.PeakMemMB,
			TotalMS:   stats.TotalMS,
		},
		CacheHit: false,
	}

	// Cache the result if caching is enabled
	if opts.UseCache {
		err = cacheRunResult(cm, intentHash, result, opts.Profile, opts.Engine, eng)
		if err != nil {
			// Log but don't fail on cache write errors
			if opts.Progress != nil {
				opts.Progress("Warning: failed to cache run result: " + err.Error())
			}
		}
	}

	// Enforce cache limits
	_ = cm.EnforceLimits("runs")

	return result, nil
}

// selectEngine selects the appropriate KCL execution engine
func selectEngine(kind EngineKind) (engine.Engine, error) {
	engineKind := engine.Kind(kind)

	// Try to get the requested engine
	eng, err := engine.Get(engineKind)
	if err != nil {
		// Try fallback selection
		eng, err = engine.SelectEngine(engineKind)
		if err != nil {
			return nil, fmt.Errorf("no suitable engine available: %w", err)
		}
	}

	return eng, nil
}

// computeIntentHash computes a deterministic hash of all execution inputs
func computeIntentHash(profile Profile, moduleDigest string, engineKind EngineKind, eng engine.Engine, valuesJSON, ctxJSON []byte) (string, error) {
	// Canonicalize JSON inputs
	canonicalValues := ""
	if len(valuesJSON) > 0 {
		canonical, err := cache.Canonicalize(valuesJSON)
		if err != nil {
			return "", fmt.Errorf("failed to canonicalize values: %w", err)
		}
		canonicalValues = string(canonical)
	}

	canonicalCtx := ""
	if len(ctxJSON) > 0 {
		canonical, err := cache.Canonicalize(ctxJSON)
		if err != nil {
			return "", fmt.Errorf("failed to canonicalize context: %w", err)
		}
		canonicalCtx = string(canonical)
	}

	// Delegate to internal.ComputeIntentHash for a single authoritative hash
	return internal.ComputeIntentHash(
		string(profile),
		moduleDigest,
		string(engineKind),
		eng.Version(),
		eng.KCLVersion(),
		canonicalValues,
		canonicalCtx,
	), nil
}

// deriveEntry determines the entry point for KCL execution
func deriveEntry(meta *ModuleMeta, profile Profile, moduleDir string) string {
	// If meta specifies entry, use it
	if meta != nil && meta.Entry != "" {
		return meta.Entry
	}

	// Common entry point patterns
	commonEntries := []string{"main.k", "index.k", "lib.k", "mod.k"}

	// Prefer profile-specific defaults and concrete files
	switch profile {
	case ProfileCompat:
		for _, entry := range commonEntries {
			entryPath := filepath.Join(moduleDir, entry)
			if fileExists(entryPath) {
				return entry
			}
		}
		// If module appears to be a KCL module (kcl.mod), try any .k file
		if fileExists(filepath.Join(moduleDir, "kcl.mod")) {
			if name := findFirstKFile(moduleDir); name != "" {
				return name
			}
		}
	case ProfileStrict:
		for _, entry := range commonEntries {
			entryPath := filepath.Join(moduleDir, entry)
			if fileExists(entryPath) {
				return entry
			}
		}
		if name := findFirstKFile(moduleDir); name != "" {
			return name
		}
	}

	// Fallback to current directory (engines may resolve this)
	return "."
}

// findFirstKFile returns the first .k file in dir or empty string
func findFirstKFile(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".k" {
			return f.Name()
		}
	}
	return ""
}

// loadCachedRun loads a cached run result if it exists and is valid
func loadCachedRun(cm *cache.Manager, intentHash, digest string, meta *ModuleMeta) (*RunResult, error) {
	yamlPath, metaPath := cm.RunPaths(intentHash)

	// Check if both files exist
	if !fileExists(yamlPath) || !fileExists(metaPath) {
		return nil, fmt.Errorf("cache miss")
	}

	// Read metadata
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var cacheMeta CacheMeta
	if err := json.Unmarshal(metaBytes, &cacheMeta); err != nil {
		return nil, err
	}

	// Check TTL
	ttl := time.Duration(cm.TTLDays) * 24 * time.Hour
	if time.Since(cacheMeta.CreatedAt) > ttl {
		// Expired, remove files
		_ = os.Remove(yamlPath)
		_ = os.Remove(metaPath)
		return nil, fmt.Errorf("cache expired")
	}

	// Read YAML output
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}

	// Convert meta to JSON
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	return &RunResult{
		YAML:     yamlBytes,
		Digest:   digest,
		MetaJSON: metaJSON,
		Stats:    cacheMeta.Stats,
		CacheHit: true,
	}, nil
}

// cacheRunResult saves a run result to the cache
func cacheRunResult(cm *cache.Manager, intentHash string, result *RunResult, profile Profile, engineKind EngineKind, eng engine.Engine) error {
	yamlPath, metaPath := cm.RunPaths(intentHash)

	// Ensure directory exists
	dir := filepath.Dir(yamlPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write YAML atomically
	if err := writeFileAtomic(yamlPath, result.YAML); err != nil {
		return fmt.Errorf("failed to write YAML: %w", err)
	}

	// Create metadata
	cacheMeta := CacheMeta{
		Digest:        result.Digest,
		Profile:       profile,
		Engine:        engineKind,
		EngineVersion: eng.Version(),
		KCLVersion:    eng.KCLVersion(),
		CreatedAt:     time.Now(),
		LastAccessAt:  time.Now(),
		Stats:         result.Stats,
	}

	metaBytes, err := json.MarshalIndent(cacheMeta, "", "  ")
	if err != nil {
		return err
	}

	// Write metadata atomically
	if err := writeFileAtomic(metaPath, metaBytes); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// touchRunCache updates the access time for a cached run
func touchRunCache(cm *cache.Manager, intentHash string) error {
	_, metaPath := cm.RunPaths(intentHash)

	// Read existing metadata
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}

	var cacheMeta CacheMeta
	if err := json.Unmarshal(metaBytes, &cacheMeta); err != nil {
		return err
	}

	// Update access time
	cacheMeta.LastAccessAt = time.Now()

	// Write back
	metaBytes, err = json.MarshalIndent(cacheMeta, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(metaPath, metaBytes)
}

// writeFileAtomic writes a file atomically using temp file and rename
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Create temp file in same directory
	temp, err := os.CreateTemp(dir, base+".tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()

	// Clean up on any error
	defer func() {
		if temp != nil {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	// Write data
	if _, err := temp.Write(data); err != nil {
		return err
	}

	// Sync to disk
	if err := temp.Sync(); err != nil {
		return err
	}

	// Close before rename
	if err := temp.Close(); err != nil {
		return err
	}
	temp = nil

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
