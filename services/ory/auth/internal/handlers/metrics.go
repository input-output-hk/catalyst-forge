package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	loginAcceptTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "auth_orchestrator",
		Name:      "login_accept_total",
		Help:      "Total number of accepted login requests",
	})
	consentAcceptTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "auth_orchestrator",
		Name:      "consent_accept_total",
		Help:      "Total number of accepted consent requests",
	})
	loginFailureTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "auth_orchestrator",
		Name:      "login_failure_total",
		Help:      "Total number of failed login handler responses",
	})
	consentFailureTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "auth_orchestrator",
		Name:      "consent_failure_total",
		Help:      "Total number of failed consent handler responses",
	})
	tokenHookFailureTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "auth_orchestrator",
		Name:      "tokenhook_failure_total",
		Help:      "Total number of failed token-hook responses",
	})
)

func init() {
	prometheus.MustRegister(loginAcceptTotal)
	prometheus.MustRegister(consentAcceptTotal)
	prometheus.MustRegister(loginFailureTotal)
	prometheus.MustRegister(consentFailureTotal)
	prometheus.MustRegister(tokenHookFailureTotal)
}
