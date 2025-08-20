package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// BuildCreatedTotal counts created builds by source (pr|merge|tag|manual).
var BuildCreatedTotal *prometheus.CounterVec

// InitDefault registers metrics to the default Prometheus registerer.
func InitDefault() {
	BuildCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "build",
			Name:      "created_total",
			Help:      "Total number of builds created.",
		},
		[]string{"source"},
	)
	CertIssuedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "cert",
			Name:      "issued_total",
			Help:      "Total number of certificates issued.",
		},
		[]string{"kind"}, // client/server
	)
	CertIssueErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "cert",
			Name:      "issue_errors_total",
			Help:      "Total number of certificate issuance errors by reason.",
		},
		[]string{"reason"},
	)
	// Removed StepCA latency metric after migration
	PCAIssueLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "foundry",
			Subsystem: "cert",
			Name:      "pca_issue_latency_seconds",
			Help:      "Latency of ACM-PCA issue/get operations.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"kind"},
	)
	// Cookie/session metrics
	SessionRefreshTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "session",
			Name:      "refresh_total",
			Help:      "Total number of session refresh attempts.",
		},
		[]string{"result"}, // success|invalid|reused
	)
	DeviceTokenModeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "device",
			Name:      "token_mode_total",
			Help:      "Total number of device token responses by mode.",
		},
		[]string{"mode"}, // cookies|cli_json
	)
	SessionLogoutTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "session",
			Name:      "logout_total",
			Help:      "Total number of session logout events.",
		},
		[]string{"result"}, // success
	)
	prometheus.MustRegister(BuildCreatedTotal, CertIssuedTotal, CertIssueErrorsTotal, PCAIssueLatencySeconds, SessionRefreshTotal, SessionLogoutTotal, DeviceTokenModeTotal)
}

// Certificate issuance metrics.
var (
	CertIssuedTotal        *prometheus.CounterVec
	CertIssueErrorsTotal   *prometheus.CounterVec
	PCAIssueLatencySeconds *prometheus.HistogramVec
	SessionRefreshTotal    *prometheus.CounterVec
	SessionLogoutTotal     *prometheus.CounterVec
	DeviceTokenModeTotal   *prometheus.CounterVec
)
