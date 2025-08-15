package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ModuleMeta contains metadata for a cached module.
type ModuleMeta struct {
	Digest      string            `json:"digest"`
	Profile     string            `json:"profile"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Sum         string            `json:"sum,omitempty"`
	Entry       string            `json:"entry,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	License     string            `json:"license,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CachedAt    time.Time         `json:"cachedAt"`
	LastAccess  time.Time         `json:"lastAccess"`
	Size        int64             `json:"size"`
}

// RunMeta contains metadata for a cached run result.
type RunMeta struct {
	IntentHash    string    `json:"intentHash"`
	ModuleDigest  string    `json:"moduleDigest"`
	Profile       string    `json:"profile"`
	Engine        string    `json:"engine"`
	EngineVersion string    `json:"engineVersion"`
	KCLVersion    string    `json:"kclVersion"`
	ValuesHash    string    `json:"valuesHash"`
	CtxHash       string    `json:"ctxHash"`
	CreatedAt     time.Time `json:"createdAt"`
	LastAccess    time.Time `json:"lastAccess"`
	Stats         RunStats  `json:"stats"`
	Size          int64     `json:"size"`
}

// RunStats contains execution statistics.
type RunStats struct {
	ColdStart bool  `json:"coldStart"`
	CompileMS int64 `json:"compileMs"`
	EvalMS    int64 `json:"evalMs"`
	PeakMemMB int   `json:"peakMemMb"`
}

// IndexMeta contains the cache index metadata.
type IndexMeta struct {
	Version      string    `json:"version"`
	LastCleaned  time.Time `json:"lastCleaned"`
	ModulesCount int       `json:"modulesCount"`
	ModulesSize  int64     `json:"modulesSize"`
	RunsCount    int       `json:"runsCount"`
	RunsSize     int64     `json:"runsSize"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// MetaStore manages cache metadata persistence.
type MetaStore struct {
	root string
}

// NewMetaStore creates a new metadata store.
func NewMetaStore(cacheRoot string) *MetaStore {
	return &MetaStore{
		root: cacheRoot,
	}
}

// SaveModuleMeta saves module metadata.
func (s *MetaStore) SaveModuleMeta(digest string, meta *ModuleMeta) error {
	metaPath := s.moduleMetaPath(digest)
	return WriteJSON(metaPath, meta)
}

// LoadModuleMeta loads module metadata.
func (s *MetaStore) LoadModuleMeta(digest string) (*ModuleMeta, error) {
	metaPath := s.moduleMetaPath(digest)
	var meta ModuleMeta
	if err := ReadJSON(metaPath, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SaveRunMeta saves run metadata.
func (s *MetaStore) SaveRunMeta(intentHash string, meta *RunMeta) error {
	metaPath := s.runMetaPath(intentHash)
	return WriteJSON(metaPath, meta)
}

// LoadRunMeta loads run metadata.
func (s *MetaStore) LoadRunMeta(intentHash string) (*RunMeta, error) {
	metaPath := s.runMetaPath(intentHash)
	var meta RunMeta
	if err := ReadJSON(metaPath, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// UpdateModuleAccess updates the last access time for a module.
func (s *MetaStore) UpdateModuleAccess(digest string) error {
	meta, err := s.LoadModuleMeta(digest)
	if err != nil {
		return err
	}
	meta.LastAccess = time.Now()
	return s.SaveModuleMeta(digest, meta)
}

// UpdateRunAccess updates the last access time for a run.
func (s *MetaStore) UpdateRunAccess(intentHash string) error {
	meta, err := s.LoadRunMeta(intentHash)
	if err != nil {
		return err
	}
	meta.LastAccess = time.Now()
	return s.SaveRunMeta(intentHash, meta)
}

// SaveIndex saves the cache index.
func (s *MetaStore) SaveIndex(index *IndexMeta) error {
	indexPath := s.indexPath()
	index.UpdatedAt = time.Now()
	return WriteJSON(indexPath, index)
}

// LoadIndex loads the cache index.
func (s *MetaStore) LoadIndex() (*IndexMeta, error) {
	indexPath := s.indexPath()
	var index IndexMeta
	if err := ReadJSON(indexPath, &index); err != nil {
		if os.IsNotExist(err) {
			// Return empty index if doesn't exist
			return &IndexMeta{
				Version:   "1.0",
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, err
	}
	return &index, nil
}

// ListModules returns a list of all cached modules.
func (s *MetaStore) ListModules() ([]*ModuleMeta, error) {
	modulesDir := filepath.Join(s.root, "modules")
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var modules []*ModuleMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		digest := entry.Name()
		meta, err := s.LoadModuleMeta(digest)
		if err != nil {
			// Skip modules without valid metadata
			continue
		}
		modules = append(modules, meta)
	}

	return modules, nil
}

// ListRuns returns a list of all cached run results.
func (s *MetaStore) ListRuns() ([]*RunMeta, error) {
	runsDir := filepath.Join(s.root, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Map to deduplicate by intent hash
	runMap := make(map[string]bool)
	var runs []*RunMeta

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		intentHash := name[:len(name)-5] // Remove .json
		if runMap[intentHash] {
			continue
		}
		runMap[intentHash] = true

		meta, err := s.LoadRunMeta(intentHash)
		if err != nil {
			// Skip runs without valid metadata
			continue
		}
		runs = append(runs, meta)
	}

	return runs, nil
}

// GetCacheStats returns cache statistics.
func (s *MetaStore) GetCacheStats() (*CacheStats, error) {
	index, err := s.LoadIndex()
	if err != nil {
		return nil, err
	}

	modules, err := s.ListModules()
	if err != nil {
		return nil, err
	}

	runs, err := s.ListRuns()
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		ModulesCount: len(modules),
		RunsCount:    len(runs),
		LastCleaned:  index.LastCleaned,
		UpdatedAt:    time.Now(),
	}

	// Calculate sizes
	for _, m := range modules {
		stats.ModulesSize += m.Size
	}
	for _, r := range runs {
		stats.RunsSize += r.Size
	}

	return stats, nil
}

// CacheStats contains cache statistics.
type CacheStats struct {
	ModulesCount int       `json:"modulesCount"`
	ModulesSize  int64     `json:"modulesSize"`
	RunsCount    int       `json:"runsCount"`
	RunsSize     int64     `json:"runsSize"`
	LastCleaned  time.Time `json:"lastCleaned"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Path helper methods

func (s *MetaStore) moduleMetaPath(digest string) string {
	return filepath.Join(s.root, "modules", digest, ".meta.json")
}

func (s *MetaStore) runMetaPath(intentHash string) string {
	return filepath.Join(s.root, "runs", intentHash+".json")
}

func (s *MetaStore) indexPath() string {
	return filepath.Join(s.root, "index", "cache.json")
}

// MigrateMetadata migrates metadata from old format to new format if needed.
func (s *MetaStore) MigrateMetadata() error {
	// This function can be used to handle metadata format changes in the future
	// For now, it's a no-op
	return nil
}

// ValidateMetadata validates the integrity of cached metadata.
func (s *MetaStore) ValidateMetadata() error {
	// Validate modules
	modules, err := s.ListModules()
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}

	for _, meta := range modules {
		if meta.Digest == "" {
			return fmt.Errorf("module metadata missing digest")
		}
		if meta.Profile == "" {
			return fmt.Errorf("module %s metadata missing profile", meta.Digest)
		}
	}

	// Validate runs
	runs, err := s.ListRuns()
	if err != nil {
		return fmt.Errorf("failed to list runs: %w", err)
	}

	for _, meta := range runs {
		if meta.IntentHash == "" {
			return fmt.Errorf("run metadata missing intent hash")
		}
		if meta.ModuleDigest == "" {
			return fmt.Errorf("run %s metadata missing module digest", meta.IntentHash)
		}
	}

	return nil
}

// CompactMetadata removes orphaned metadata entries.
func (s *MetaStore) CompactMetadata() error {
	// This would remove metadata for modules/runs that no longer exist
	// Implementation depends on the specific requirements
	return nil
}

// ExportMetadata exports all metadata to a JSON file.
func (s *MetaStore) ExportMetadata(path string) error {
	export := struct {
		Version  string        `json:"version"`
		Exported time.Time     `json:"exported"`
		Modules  []*ModuleMeta `json:"modules"`
		Runs     []*RunMeta    `json:"runs"`
		Index    *IndexMeta    `json:"index"`
	}{
		Version:  "1.0",
		Exported: time.Now(),
	}

	var err error
	export.Modules, err = s.ListModules()
	if err != nil {
		return err
	}

	export.Runs, err = s.ListRuns()
	if err != nil {
		return err
	}

	export.Index, err = s.LoadIndex()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}