package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the KCL library.
type Metrics struct {
	// Counters
	ModulePulls      prometheus.Counter
	ModuleVerifies   prometheus.Counter
	ModulePublishes  prometheus.Counter
	RunExecutions    prometheus.Counter
	CacheHits        prometheus.Counter
	CacheMisses      prometheus.Counter
	CacheEvictions   prometheus.Counter
	
	// Gauges
	CacheModulesSize prometheus.Gauge
	CacheRunsSize    prometheus.Gauge
	CacheModulesCount prometheus.Gauge
	CacheRunsCount   prometheus.Gauge
	
	// Histograms
	PullDuration    prometheus.Histogram
	VerifyDuration  prometheus.Histogram
	PublishDuration prometheus.Histogram
	RunDuration     prometheus.Histogram
	PackDuration    prometheus.Histogram
	
	// Summary
	CacheHitRate    prometheus.Summary
}

var (
	metrics     *Metrics
	metricsOnce sync.Once
)

// GetMetrics returns the singleton metrics instance.
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = initMetrics()
	})
	return metrics
}

// initMetrics initializes all Prometheus metrics.
func initMetrics() *Metrics {
	return &Metrics{
		// Counters
		ModulePulls: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "module_pulls_total",
			Help:      "Total number of module pulls",
		}),
		ModuleVerifies: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "module_verifies_total",
			Help:      "Total number of module verifications",
		}),
		ModulePublishes: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "module_publishes_total",
			Help:      "Total number of module publishes",
		}),
		RunExecutions: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "run_executions_total",
			Help:      "Total number of KCL run executions",
		}),
		CacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		}),
		CacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		}),
		CacheEvictions: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_evictions_total",
			Help:      "Total number of cache evictions",
		}),
		
		// Gauges
		CacheModulesSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_modules_size_bytes",
			Help:      "Current size of module cache in bytes",
		}),
		CacheRunsSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_runs_size_bytes",
			Help:      "Current size of run cache in bytes",
		}),
		CacheModulesCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_modules_count",
			Help:      "Current number of cached modules",
		}),
		CacheRunsCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "cache_runs_count",
			Help:      "Current number of cached runs",
		}),
		
		// Histograms
		PullDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "pull_duration_seconds",
			Help:      "Duration of module pull operations",
			Buckets:   prometheus.DefBuckets,
		}),
		VerifyDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "verify_duration_seconds",
			Help:      "Duration of module verify operations",
			Buckets:   prometheus.DefBuckets,
		}),
		PublishDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "publish_duration_seconds",
			Help:      "Duration of module publish operations",
			Buckets:   prometheus.DefBuckets,
		}),
		RunDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "run_duration_seconds",
			Help:      "Duration of KCL run operations",
			Buckets:   prometheus.DefBuckets,
		}),
		PackDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "forge",
			Subsystem: "kcl",
			Name:      "pack_duration_seconds",
			Help:      "Duration of module pack operations",
			Buckets:   prometheus.DefBuckets,
		}),
		
		// Summary
		CacheHitRate: promauto.NewSummary(prometheus.SummaryOpts{
			Namespace:  "forge",
			Subsystem:  "kcl",
			Name:       "cache_hit_rate",
			Help:       "Cache hit rate as a percentage",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}
}

// ObserveOperation measures the duration of an operation.
func ObserveOperation(ctx context.Context, histogram prometheus.Histogram, fn func() error) error {
	start := time.Now()
	err := fn()
	histogram.Observe(time.Since(start).Seconds())
	return err
}

// UpdateCacheMetrics updates cache-related metrics.
func UpdateCacheMetrics(moduleCount, runCount int, moduleSize, runSize int64) {
	m := GetMetrics()
	m.CacheModulesCount.Set(float64(moduleCount))
	m.CacheRunsCount.Set(float64(runCount))
	m.CacheModulesSize.Set(float64(moduleSize))
	m.CacheRunsSize.Set(float64(runSize))
}

// RecordCacheHit records a cache hit.
func RecordCacheHit() {
	GetMetrics().CacheHits.Inc()
	updateHitRate(true)
}

// RecordCacheMiss records a cache miss.
func RecordCacheMiss() {
	GetMetrics().CacheMisses.Inc()
	updateHitRate(false)
}

// updateHitRate updates the cache hit rate summary.
func updateHitRate(hit bool) {
	value := 0.0
	if hit {
		value = 1.0
	}
	GetMetrics().CacheHitRate.Observe(value)
}

// RecordEviction records a cache eviction.
func RecordEviction(count int) {
	GetMetrics().CacheEvictions.Add(float64(count))
}