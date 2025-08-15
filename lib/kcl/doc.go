// Package kcl provides functionality to package, publish, verify, pull, cache, and execute
// KCL modules distributed as OCI artifacts. It offers deterministic caching of both module
// contents (by digest) and run outputs (by intent hash).
//
// The package supports two profiles:
//   - ProfileCompat: KPM-compatible with single tar layer
//   - ProfileStrict: Forge-specific with tar + meta.json and CUE validation
//
// It includes two execution engines:
//   - EngineNative: CGO-based kcl-go for high performance
//   - EngineWASM: Sandboxed WASM runtime for isolation
//
// Caching:
//
// The package provides two levels of caching:
//   - Module cache: Extracted modules stored by digest
//   - Run cache: KCL evaluation results keyed by intent hash
//
// The intent hash is computed from:
//   - Module digest
//   - Canonicalized input values and context
//   - Engine type and version
//   - KCL runtime version
//
// Cache location defaults to ~/.forge/kcl/ and can be configured via environment variables.
// Eviction uses size-bounded LRU with TTL expiration.
//
// Concurrency:
//
// All operations are safe for concurrent use within and across processes using:
//   - Singleflight for in-process deduplication
//   - File system locks for cross-process coordination
package kcl