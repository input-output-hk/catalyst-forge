package kcl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/cache"
)

// CacheStats contains cache statistics.
type CacheStats struct {
	CacheDir     string         `json:"cacheDir"`
	ModulesStats ModuleStats    `json:"modules"`
	RunsStats    RunStats       `json:"runs"`
	TotalSize    int64          `json:"totalSize"`
	LastCleaned  time.Time      `json:"lastCleaned,omitempty"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// ModuleStats contains module cache statistics.
type ModuleStats struct {
	Count        int           `json:"count"`
	TotalSize    int64         `json:"totalSize"`
	MaxSize      int64         `json:"maxSize"`
	OldestAccess time.Time     `json:"oldestAccess,omitempty"`
	NewestAccess time.Time     `json:"newestAccess,omitempty"`
	HitRate      float64       `json:"hitRate,omitempty"`
	TopModules   []ModuleUsage `json:"topModules,omitempty"`
}

// RunStats contains run cache statistics.
type RunStats struct {
	Count        int         `json:"count"`
	TotalSize    int64       `json:"totalSize"`
	MaxSize      int64       `json:"maxSize"`
	OldestAccess time.Time   `json:"oldestAccess,omitempty"`
	NewestAccess time.Time   `json:"newestAccess,omitempty"`
	HitRate      float64     `json:"hitRate,omitempty"`
	TopRuns      []RunUsage  `json:"topRuns,omitempty"`
}

// ModuleUsage tracks usage of a cached module.
type ModuleUsage struct {
	Digest     string    `json:"digest"`
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Size       int64     `json:"size"`
	AccessCount int      `json:"accessCount"`
	LastAccess time.Time `json:"lastAccess"`
}

// RunUsage tracks usage of a cached run.
type RunUsage struct {
	IntentHash   string    `json:"intentHash"`
	ModuleDigest string    `json:"moduleDigest"`
	Size         int64     `json:"size"`
	AccessCount  int       `json:"accessCount"`
	LastAccess   time.Time `json:"lastAccess"`
}

// GetCacheStats returns comprehensive cache statistics.
func GetCacheStats() (*CacheStats, error) {
	cm, err := cache.GetManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache manager: %w", err)
	}

	stats := &CacheStats{
		CacheDir:  cm.Root,
		UpdatedAt: time.Now(),
	}

	// Get module stats
	moduleStats, err := getModuleStats(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to get module stats: %w", err)
	}
	stats.ModulesStats = *moduleStats

	// Get run stats
	runStats, err := getRunStats(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to get run stats: %w", err)
	}
	stats.RunsStats = *runStats

	// Calculate total size
	stats.TotalSize = stats.ModulesStats.TotalSize + stats.RunsStats.TotalSize

	// Get last cleaned time
	indexPath := filepath.Join(cm.Root, "index", "cache.json")
	if data, err := os.ReadFile(indexPath); err == nil {
		var index map[string]interface{}
		if err := json.Unmarshal(data, &index); err == nil {
			if lastCleaned, ok := index["lastCleaned"].(string); ok {
				if t, err := time.Parse(time.RFC3339, lastCleaned); err == nil {
					stats.LastCleaned = t
				}
			}
		}
	}

	return stats, nil
}

// getModuleStats calculates module cache statistics.
func getModuleStats(cm *cache.Manager) (*ModuleStats, error) {
	stats := &ModuleStats{
		MaxSize: cm.ModulesMaxBytes,
	}

	modulesDir := filepath.Join(cm.Root, "modules")
	if !dirExists(modulesDir) {
		return stats, nil
	}

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, err
	}

	var oldest, newest time.Time
	topModules := make([]ModuleUsage, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		digest := entry.Name()
		moduleDir := filepath.Join(modulesDir, digest)
		
		// Get module size
		size, err := getDirSize(moduleDir)
		if err != nil {
			continue
		}

		// Read metadata
		metaPath := filepath.Join(moduleDir, ".meta.json")
		var usage ModuleUsage
		usage.Digest = digest
		usage.Size = size

		if data, err := os.ReadFile(metaPath); err == nil {
			var meta ModuleMeta
			if err := json.Unmarshal(data, &meta); err == nil {
				usage.Name = meta.Name
				usage.Version = meta.Version
			}
		}

		// Get access time from stamp file
		stampPath := filepath.Join(moduleDir, ".stamp")
		if info, err := os.Stat(stampPath); err == nil {
			usage.LastAccess = info.ModTime()
			
			if oldest.IsZero() || usage.LastAccess.Before(oldest) {
				oldest = usage.LastAccess
			}
			if usage.LastAccess.After(newest) {
				newest = usage.LastAccess
			}
		}

		stats.Count++
		stats.TotalSize += size
		topModules = append(topModules, usage)
	}

	stats.OldestAccess = oldest
	stats.NewestAccess = newest

	// Sort and limit top modules
	if len(topModules) > 10 {
		stats.TopModules = topModules[:10]
	} else {
		stats.TopModules = topModules
	}

	return stats, nil
}

// getRunStats calculates run cache statistics.
func getRunStats(cm *cache.Manager) (*RunStats, error) {
	stats := &RunStats{
		MaxSize: cm.RunsMaxBytes,
	}

	runsDir := filepath.Join(cm.Root, "runs")
	if !dirExists(runsDir) {
		return stats, nil
	}

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, err
	}

	var oldest, newest time.Time
	runMap := make(map[string]*RunUsage)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) == ".yaml" {
			intentHash := name[:len(name)-5]
			
			if _, exists := runMap[intentHash]; !exists {
				runMap[intentHash] = &RunUsage{
					IntentHash: intentHash,
				}
			}

			// Get file size
			filePath := filepath.Join(runsDir, name)
			if info, err := os.Stat(filePath); err == nil {
				runMap[intentHash].Size += info.Size()
				runMap[intentHash].LastAccess = info.ModTime()

				if oldest.IsZero() || info.ModTime().Before(oldest) {
					oldest = info.ModTime()
				}
				if info.ModTime().After(newest) {
					newest = info.ModTime()
				}
			}
		} else if filepath.Ext(name) == ".json" {
			intentHash := name[:len(name)-5]
			
			// Read metadata
			metaPath := filepath.Join(runsDir, name)
			if data, err := os.ReadFile(metaPath); err == nil {
				var meta map[string]interface{}
				if err := json.Unmarshal(data, &meta); err == nil {
					if digest, ok := meta["moduleDigest"].(string); ok && runMap[intentHash] != nil {
						runMap[intentHash].ModuleDigest = digest
					}
				}
			}
		}
	}

	stats.OldestAccess = oldest
	stats.NewestAccess = newest

	// Convert map to slice
	topRuns := make([]RunUsage, 0, len(runMap))
	for _, usage := range runMap {
		stats.Count++
		stats.TotalSize += usage.Size
		topRuns = append(topRuns, *usage)
	}

	// Sort and limit top runs
	if len(topRuns) > 10 {
		stats.TopRuns = topRuns[:10]
	} else {
		stats.TopRuns = topRuns
	}

	return stats, nil
}

// CleanCache removes expired and least recently used cache entries.
func CleanCache(dryRun bool) (*CleanResult, error) {
	cm, err := cache.GetManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache manager: %w", err)
	}

	result := &CleanResult{
		StartedAt: time.Now(),
		DryRun:    dryRun,
	}

	// Clean modules
	if err := cleanModules(cm, result, dryRun); err != nil {
		return nil, fmt.Errorf("failed to clean modules: %w", err)
	}

	// Clean runs
	if err := cleanRuns(cm, result, dryRun); err != nil {
		return nil, fmt.Errorf("failed to clean runs: %w", err)
	}

	result.EndedAt = time.Now()
	result.Duration = result.EndedAt.Sub(result.StartedAt)

	// Update index with last cleaned time
	if !dryRun {
		indexPath := filepath.Join(cm.Root, "index", "cache.json")
		index := map[string]interface{}{
			"lastCleaned": time.Now(),
			"version":     "1.0",
		}
		if data, err := json.MarshalIndent(index, "", "  "); err == nil {
			_ = os.MkdirAll(filepath.Dir(indexPath), 0755)
			_ = os.WriteFile(indexPath, data, 0644)
		}
	}

	return result, nil
}

// CleanResult contains the result of a cache clean operation.
type CleanResult struct {
	DryRun         bool          `json:"dryRun"`
	ModulesRemoved int           `json:"modulesRemoved"`
	RunsRemoved    int           `json:"runsRemoved"`
	SpaceFreed     int64         `json:"spaceFreed"`
	StartedAt      time.Time     `json:"startedAt"`
	EndedAt        time.Time     `json:"endedAt"`
	Duration       time.Duration `json:"duration"`
	Errors         []string      `json:"errors,omitempty"`
}

// cleanModules cleans expired module cache entries.
func cleanModules(cm *cache.Manager, result *CleanResult, dryRun bool) error {
	// Implementation would check TTL and LRU eviction
	// For now, this is a placeholder
	return nil
}

// cleanRuns cleans expired run cache entries.
func cleanRuns(cm *cache.Manager, result *CleanResult, dryRun bool) error {
	// Implementation would check TTL and LRU eviction
	// For now, this is a placeholder
	return nil
}

// Helper functions

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

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