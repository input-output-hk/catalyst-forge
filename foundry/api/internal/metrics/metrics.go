package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// BuildSessionCreated counts created build sessions labeled by owner_type.
var BuildSessionCreated *prometheus.CounterVec

// InitDefault registers metrics to the default Prometheus registerer.
func InitDefault() {
	BuildSessionCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "foundry",
			Subsystem: "build",
			Name:      "session_created_total",
			Help:      "Total number of build sessions created.",
		},
		[]string{"owner_type"},
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
	prometheus.MustRegister(BuildSessionCreated, CertIssuedTotal, CertIssueErrorsTotal, PCAIssueLatencySeconds, SessionRefreshTotal, SessionLogoutTotal, DeviceTokenModeTotal)
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
