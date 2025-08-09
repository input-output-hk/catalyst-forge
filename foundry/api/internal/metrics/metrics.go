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
	StepCASignLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "foundry",
			Subsystem: "cert",
			Name:      "stepca_sign_latency_seconds",
			Help:      "Latency of step-ca sign requests.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"kind"},
	)
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
	prometheus.MustRegister(BuildSessionCreated, CertIssuedTotal, CertIssueErrorsTotal, StepCASignLatencySeconds, PCAIssueLatencySeconds)
}

// Certificate issuance metrics
var (
	CertIssuedTotal          *prometheus.CounterVec
	CertIssueErrorsTotal     *prometheus.CounterVec
	StepCASignLatencySeconds *prometheus.HistogramVec
	PCAIssueLatencySeconds   *prometheus.HistogramVec
)
