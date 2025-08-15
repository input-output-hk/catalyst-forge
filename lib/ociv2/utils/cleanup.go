package utils

import (
	"io"
	"sync"
)

// cleanupManager ensures proper resource cleanup
type cleanupManager struct {
	cleanups []func()
	mu       sync.Mutex
}

// newCleanupManager creates a new cleanup manager
func newCleanupManager() *cleanupManager {
	return &cleanupManager{
		cleanups: make([]func(), 0),
	}
}

// Add registers a cleanup function
func (cm *cleanupManager) Add(fn func()) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cleanups = append(cm.cleanups, fn)
}

// AddCloser registers an io.Closer for cleanup
func (cm *cleanupManager) AddCloser(c io.Closer) {
	if c != nil {
		cm.Add(func() {
			_ = c.Close() // Ignore close errors in cleanup
		})
	}
}

// Cleanup runs all registered cleanup functions
func (cm *cleanupManager) Cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Run cleanups in reverse order (LIFO)
	for i := len(cm.cleanups) - 1; i >= 0; i-- {
		if cm.cleanups[i] != nil {
			cm.cleanups[i]()
		}
	}
	
	// Clear the list
	cm.cleanups = cm.cleanups[:0]
}

// safeClose safely closes a resource, handling nil and errors
func safeClose(c io.Closer) error {
	if c == nil {
		return nil
	}
	return c.Close()
}

// withCleanup executes a function with automatic cleanup
func withCleanup(fn func(*cleanupManager) error) error {
	cm := newCleanupManager()
	defer cm.Cleanup()
	return fn(cm)
}

// resourceTracker tracks and ensures cleanup of resources
type resourceTracker struct {
	resources map[string]io.Closer
	mu        sync.RWMutex
}

// newResourceTracker creates a new resource tracker
func newResourceTracker() *resourceTracker {
	return &resourceTracker{
		resources: make(map[string]io.Closer),
	}
}

// Track registers a resource for tracking
func (rt *resourceTracker) Track(key string, resource io.Closer) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	// Close any existing resource with the same key
	if existing, ok := rt.resources[key]; ok {
		_ = existing.Close()
	}
	
	rt.resources[key] = resource
}

// Release releases and closes a specific resource
func (rt *resourceTracker) Release(key string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	if resource, ok := rt.resources[key]; ok {
		delete(rt.resources, key)
		return safeClose(resource)
	}
	
	return nil
}

// ReleaseAll releases and closes all tracked resources
func (rt *resourceTracker) ReleaseAll() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	for key, resource := range rt.resources {
		_ = safeClose(resource)
		delete(rt.resources, key)
	}
}

// ensureCleanup ensures a cleanup function is called even on panic
func ensureCleanup(cleanup func()) {
	if r := recover(); r != nil {
		cleanup()
		panic(r) // Re-panic after cleanup
	}
	cleanup()
}