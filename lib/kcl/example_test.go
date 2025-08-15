package kcl_test

import (
	"context"
	"fmt"
	"log"

	"github.com/input-output-hk/catalyst-forge/lib/kcl"
)

func ExampleRun() {
	// Create an OCI client (from lib/ociv2)
	var ociClient kcl.OCI // This would be an actual OCI client instance
	
	// Define the module to run
	ref := kcl.ModuleRef{
		Repo: "oci://ghcr.io/example/my-kcl-module",
		Tag:  "v1.0.0",
	}
	
	// Configure run options
	opts := kcl.RunOptions{
		Profile: kcl.ProfileCompat,     // Use KPM-compatible profile
		Engine:  kcl.EngineNative,      // Use native CGO engine
		Values: []byte(`{
			"app_name": "my-app",
			"replicas": 3,
			"image": "nginx:latest"
		}`),
		Context: []byte(`{
			"environment": "production",
			"region": "us-west-2"
		}`),
		TimeoutSec:     60,   // 60 second timeout
		UseCache:       true, // Enable caching
		ForceRecompute: false,
	}
	
	// Run the KCL module
	ctx := context.Background()
	result, err := kcl.Run(ctx, ociClient, ref, opts)
	if err != nil {
		log.Fatal(err)
	}
	
	// Check if result was from cache
	if result.CacheHit {
		fmt.Println("Result retrieved from cache")
	} else {
		fmt.Println("Result computed and cached")
	}
	
	// Use the YAML output
	fmt.Printf("Output YAML:\n%s\n", result.YAML)
	
	// Access execution statistics
	fmt.Printf("Execution took %dms\n", result.Stats.TotalMS)
	fmt.Printf("Peak memory: %dMB\n", result.Stats.PeakMemMB)
}

func ExampleRun_withDifferentEngines() {
	var ociClient kcl.OCI
	ref := kcl.ModuleRef{
		Repo: "oci://ghcr.io/example/my-kcl-module",
		Tag:  "v1.0.0",
	}
	
	// Use native engine for development (faster)
	devOpts := kcl.RunOptions{
		Profile:   kcl.ProfileCompat,
		Engine:    kcl.EngineNative,
		UseCache:  true,
		Values:    []byte(`{"debug": true}`),
	}
	
	// Use WASM engine for production (sandboxed)
	prodOpts := kcl.RunOptions{
		Profile:       kcl.ProfileCompat,
		Engine:        kcl.EngineWASM,
		UseCache:      true,
		MemoryLimitMB: 256,  // Limit WASM memory
		TimeoutSec:    30,   // Strict timeout
		Values:        []byte(`{"debug": false}`),
	}
	
	ctx := context.Background()
	
	// Run in development
	devResult, err := kcl.Run(ctx, ociClient, ref, devOpts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Dev result (native): %s\n", devResult.YAML)
	
	// Run in production
	prodResult, err := kcl.Run(ctx, ociClient, ref, prodOpts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Prod result (WASM): %s\n", prodResult.YAML)
}

func ExampleRun_cacheInvalidation() {
	var ociClient kcl.OCI
	ref := kcl.ModuleRef{
		Repo: "oci://ghcr.io/example/my-kcl-module",
		Tag:  "v1.0.0",
	}
	
	opts := kcl.RunOptions{
		Profile:  kcl.ProfileCompat,
		Engine:   kcl.EngineNative,
		UseCache: true,
		Values:   []byte(`{"version": "1.0"}`),
	}
	
	ctx := context.Background()
	
	// First run - will compute and cache
	result1, _ := kcl.Run(ctx, ociClient, ref, opts)
	fmt.Printf("First run cached: %v\n", !result1.CacheHit)
	
	// Second run - will use cache
	result2, _ := kcl.Run(ctx, ociClient, ref, opts)
	fmt.Printf("Second run from cache: %v\n", result2.CacheHit)
	
	// Force recompute - will bypass cache but update it
	opts.ForceRecompute = true
	result3, _ := kcl.Run(ctx, ociClient, ref, opts)
	fmt.Printf("Forced recompute: %v\n", !result3.CacheHit)
	
	// Change values - will invalidate cache
	opts.ForceRecompute = false
	opts.Values = []byte(`{"version": "2.0"}`)
	result4, _ := kcl.Run(ctx, ociClient, ref, opts)
	fmt.Printf("Different values cached: %v\n", !result4.CacheHit)
}