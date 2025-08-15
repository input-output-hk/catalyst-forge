package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Manager manages the KCL cache with LRU eviction and TTL.
type Manager struct {
	Root            string // Base cache directory
	ModulesMaxBytes int64  // Max size for module cache
	RunsMaxBytes    int64  // Max size for run cache
	TTLDays         int    // TTL for cache entries
	EnableBlobCache bool   // Whether to cache raw blobs

	// Singleflight groups for in-process deduplication
	gfModules singleflight.Group
	gfRuns    singleflight.Group

	// Mutex for cache operations
	mu sync.RWMutex

	// Track cache sizes (lazily computed)
	modulesSize int64
	runsSize    int64
}

// singleton instance for the default cache manager
var (
	defaultManager *Manager
	managerOnce    sync.Once
	managerErr     error
)

// GetManager returns the default cache manager instance.
func GetManager() (*Manager, error) {
	managerOnce.Do(func() {
		// Read from environment variables
		root := os.Getenv("FORGE_KCL_CACHE_DIR")
		if root == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				managerErr = fmt.Errorf("failed to get home directory: %w", err)
				return
			}
			root = filepath.Join(home, ".forge", "kcl")
		}

		modulesMax := int64(10 * 1024 * 1024 * 1024) // 10 GiB default
		if s := os.Getenv("FORGE_KCL_MODULE_CACHE_MAX_BYTES"); s != "" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				modulesMax = v
			}
		}

		runsMax := int64(5 * 1024 * 1024 * 1024) // 5 GiB default
		if s := os.Getenv("FORGE_KCL_RUN_CACHE_MAX_BYTES"); s != "" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				runsMax = v
			}
		}

		ttlDays := 30 // 30 days default
		if s := os.Getenv("FORGE_KCL_CACHE_TTL_DAYS"); s != "" {
			if v, err := strconv.Atoi(s); err == nil {
				ttlDays = v
			}
		}

		enableBlobs := false
		if s := os.Getenv("FORGE_KCL_ENABLE_BLOB_CACHE"); s != "" {
			enableBlobs = s == "true" || s == "1"
		}

		defaultManager, managerErr = NewManager(root, modulesMax, runsMax, ttlDays, enableBlobs)
	})

	return defaultManager, managerErr
}

// NewManager creates a new cache manager with the given options.
func NewManager(root string, modulesMax, runsMax int64, ttlDays int, enableBlobs bool) (*Manager, error) {
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		root = filepath.Join(home, ".forge", "kcl")
	}

	m := &Manager{
		Root:            root,
		ModulesMaxBytes: modulesMax,
		RunsMaxBytes:    runsMax,
		TTLDays:         ttlDays,
		EnableBlobCache: enableBlobs,
	}

	// Create cache directories
	dirs := []string{
		m.modulesDir(),
		m.blobsDir(),
		m.runsDir(),
		m.locksDir(),
		m.indexDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory %s: %w", dir, err)
		}
	}

	// Initialize cache sizes
	if err := m.computeCacheSizes(); err != nil {
		return nil, fmt.Errorf("failed to compute cache sizes: %w", err)
	}

	return m, nil
}

// Path helper methods

func (m *Manager) modulesDir() string {
	return filepath.Join(m.Root, "modules")
}

func (m *Manager) blobsDir() string {
	return filepath.Join(m.Root, "blobs")
}

func (m *Manager) runsDir() string {
	return filepath.Join(m.Root, "runs")
}

func (m *Manager) locksDir() string {
	return filepath.Join(m.Root, "locks")
}

func (m *Manager) indexDir() string {
	return filepath.Join(m.Root, "index")
}

// ModulePath returns the path for a cached module.
func (m *Manager) ModulePath(digest string) string {
	return filepath.Join(m.modulesDir(), digest)
}

// BlobPath returns the path for a cached blob.
func (m *Manager) BlobPath(digest string) string {
	return filepath.Join(m.blobsDir(), digest+".tar")
}

// RunPaths returns the paths for cached run output and metadata.
func (m *Manager) RunPaths(intentHash string) (yamlPath, metaPath string) {
	yamlPath = filepath.Join(m.runsDir(), intentHash+".yaml")
	metaPath = filepath.Join(m.runsDir(), intentHash+".json")
	return
}

// LockPath returns the path for a lock file.
func (m *Manager) LockPath(kind, key string) string {
	return filepath.Join(m.locksDir(), fmt.Sprintf("%s-%s.lock", kind, key))
}

// WithModuleLock executes a function with a module lock held.
func (m *Manager) WithModuleLock(digest string, fn func() error) error {
	lockPath := m.LockPath("modules", digest)
	lock, err := AcquireLock(lockPath)
	if err != nil {
		return fmt.Errorf("failed to acquire module lock: %w", err)
	}
	defer func() {
		_ = lock.Release()
	}()
	return fn()
}

// WithRunLock executes a function with a run lock held.
func (m *Manager) WithRunLock(intentHash string, fn func() error) error {
	lockPath := m.LockPath("runs", intentHash)
	lock, err := AcquireLock(lockPath)
	if err != nil {
		return fmt.Errorf("failed to acquire run lock: %w", err)
	}
	defer func() {
		_ = lock.Release()
	}()
	return fn()
}

// SingleflightModule deduplicates concurrent module operations.
func (m *Manager) SingleflightModule(key string, fn func() (interface{}, error)) (interface{}, error) {
	v, err, _ := m.gfModules.Do(key, fn)
	return v, err
}

// SingleflightRun deduplicates concurrent run operations.
func (m *Manager) SingleflightRun(key string, fn func() (interface{}, error)) (interface{}, error) {
	v, err, _ := m.gfRuns.Do(key, fn)
	return v, err
}

// Cache size management

func (m *Manager) computeCacheSizes() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var modulesSize, runsSize int64

	// Calculate modules size
	modulesPath := m.modulesDir()
	if err := filepath.Walk(modulesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			modulesSize += info.Size()
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk modules directory: %w", err)
	}

	// Calculate runs size
	runsPath := m.runsDir()
	if err := filepath.Walk(runsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			runsSize += info.Size()
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk runs directory: %w", err)
	}

	m.modulesSize = modulesSize
	m.runsSize = runsSize
	return nil
}

// EnforceLimits enforces size limits for the specified cache kind.
func (m *Manager) EnforceLimits(kind string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch kind {
	case "modules":
		return m.enforceModulesLimit()
	case "runs":
		return m.enforceRunsLimit()
	default:
		return fmt.Errorf("unknown cache kind: %s", kind)
	}
}

func (m *Manager) enforceModulesLimit() error {
	if m.modulesSize <= m.ModulesMaxBytes {
		return nil
	}

	// Get all module entries with access times
	entries, err := m.getModuleEntries()
	if err != nil {
		return err
	}

	// Sort by last access time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastAccess.Before(entries[j].LastAccess)
	})

	// Remove oldest entries until under limit
	for _, entry := range entries {
		if m.modulesSize <= m.ModulesMaxBytes {
			break
		}

		entryPath := m.ModulePath(entry.Digest)
		size, err := getDirSize(entryPath)
		if err != nil {
			continue
		}

		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove module %s: %w", entry.Digest, err)
		}

		m.modulesSize -= size
	}

	return nil
}

func (m *Manager) enforceRunsLimit() error {
	if m.runsSize <= m.RunsMaxBytes {
		return nil
	}

	// Get all run entries with access times
	entries, err := m.getRunEntries()
	if err != nil {
		return err
	}

	// Sort by last access time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastAccess.Before(entries[j].LastAccess)
	})

	// Remove oldest entries until under limit
	for _, entry := range entries {
		if m.runsSize <= m.RunsMaxBytes {
			break
		}

		yamlPath, metaPath := m.RunPaths(entry.IntentHash)
		
		yamlSize := getFileSize(yamlPath)
		metaSize := getFileSize(metaPath)
		
		_ = os.Remove(yamlPath)
		_ = os.Remove(metaPath)
		
		m.runsSize -= (yamlSize + metaSize)
	}

	return nil
}

// CacheEntry represents a cache entry with metadata.
type CacheEntry struct {
	Digest     string    `json:"digest,omitempty"`
	IntentHash string    `json:"intentHash,omitempty"`
	LastAccess time.Time `json:"lastAccess"`
	Size       int64     `json:"size"`
}

func (m *Manager) getModuleEntries() ([]CacheEntry, error) {
	var entries []CacheEntry
	
	modulesPath := m.modulesDir()
	files, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		digest := file.Name()
		modulePath := filepath.Join(modulesPath, digest)
		stampPath := filepath.Join(modulePath, ".stamp")
		
		// Get last access time
		var lastAccess time.Time
		if info, err := os.Stat(stampPath); err == nil {
			lastAccess = info.ModTime()
		} else {
			// Fallback to directory mod time
			if info, err := os.Stat(modulePath); err == nil {
				lastAccess = info.ModTime()
			}
		}

		size, _ := getDirSize(modulePath)
		
		entries = append(entries, CacheEntry{
			Digest:     digest,
			LastAccess: lastAccess,
			Size:       size,
		})
	}

	return entries, nil
}

func (m *Manager) getRunEntries() ([]CacheEntry, error) {
	var entries []CacheEntry
	
	runsPath := m.runsDir()
	files, err := os.ReadDir(runsPath)
	if err != nil {
		return nil, err
	}

	// Group by intent hash
	intentMap := make(map[string]CacheEntry)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		var intentHash string
		
		if filepath.Ext(name) == ".yaml" {
			intentHash = name[:len(name)-5]
		} else if filepath.Ext(name) == ".json" {
			intentHash = name[:len(name)-5]
		} else {
			continue
		}

		if entry, exists := intentMap[intentHash]; exists {
			// Update with newer access time if needed
			filePath := filepath.Join(runsPath, name)
			if info, err := os.Stat(filePath); err == nil {
				if info.ModTime().After(entry.LastAccess) {
					entry.LastAccess = info.ModTime()
				}
				entry.Size += info.Size()
				intentMap[intentHash] = entry
			}
		} else {
			filePath := filepath.Join(runsPath, name)
			if info, err := os.Stat(filePath); err == nil {
				intentMap[intentHash] = CacheEntry{
					IntentHash: intentHash,
					LastAccess: info.ModTime(),
					Size:       info.Size(),
				}
			}
		}
	}

	for _, entry := range intentMap {
		entries = append(entries, entry)
	}

	return entries, nil
}

// TouchStamp updates the access timestamp for a cache entry.
func (m *Manager) TouchStamp(path string) error {
	stampPath := filepath.Join(path, ".stamp")
	
	// Create or update stamp file
	file, err := os.Create(stampPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	
	// Write current timestamp
	_, err = file.WriteString(strconv.FormatInt(time.Now().Unix(), 10))
	return err
}

// IsExpired checks if a cache entry is expired based on TTL.
func (m *Manager) IsExpired(path string, ttlDays int) bool {
	if ttlDays <= 0 {
		return false // No expiration
	}

	info, err := os.Stat(path)
	if err != nil {
		return true // Treat as expired if can't stat
	}

	age := time.Since(info.ModTime())
	return age > time.Duration(ttlDays)*24*time.Hour
}

// Clean removes expired entries from the cache.
func (m *Manager) Clean() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean modules
	if err := m.cleanModules(); err != nil {
		return fmt.Errorf("failed to clean modules: %w", err)
	}

	// Clean runs
	if err := m.cleanRuns(); err != nil {
		return fmt.Errorf("failed to clean runs: %w", err)
	}

	// Recompute sizes
	return m.computeCacheSizes()
}

func (m *Manager) cleanModules() error {
	entries, err := m.getModuleEntries()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		modulePath := m.ModulePath(entry.Digest)
		if m.IsExpired(modulePath, m.TTLDays) {
			_ = os.RemoveAll(modulePath)
		}
	}

	return nil
}

func (m *Manager) cleanRuns() error {
	entries, err := m.getRunEntries()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		yamlPath, metaPath := m.RunPaths(entry.IntentHash)
		if m.IsExpired(metaPath, m.TTLDays) {
			_ = os.Remove(yamlPath)
			_ = os.Remove(metaPath)
		}
	}

	return nil
}

// Helper functions

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// WriteAtomically writes data to a file atomically.
func WriteAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to temp file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}

	// Sync to disk
	file, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	_ = file.Close()

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// ReadJSON reads and unmarshals a JSON file.
func ReadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// WriteJSON marshals and writes a JSON file atomically.
func WriteJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return WriteAtomically(path, data, 0644)
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}