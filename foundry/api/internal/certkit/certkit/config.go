package certkit

import "time"

// Config defines configuration for the certificate issuance subsystem.
type Config struct {
	// Required
	CAArn  string
	Region string

	// PCA defaults (can be overridden per request)
	DefaultTemplateArn string
	DefaultSigningAlgo string
	DefaultTTL         time.Duration

	// Validation / policy (defense in depth)
	MaxTTL            time.Duration
	AllowedTemplates  map[string]string // friendly key → template ARN
	AllowedKeyAlgos   []string          // e.g. ["RSA","ECDSA"]
	AllowedKeySizes   []int             // e.g. [2048,3072,4096,256]
	AllowedSANDomains []string          // e.g. [".projectcatalyst.io"]
	AllowURISAN       bool
	AllowIPSAN        bool

	// Behavior
	PollInterval time.Duration // default ~500ms
	MaxWait      time.Duration // default ~30s
	RateEnabled  bool
}

// DefaultConfig returns sane defaults; callers should override as needed.
func DefaultConfig() Config {
	return Config{
		DefaultSigningAlgo: "SHA256WITHRSA",
		DefaultTTL:         90 * 24 * time.Hour,
		MaxTTL:             365 * 24 * time.Hour,
		PollInterval:       500 * time.Millisecond,
		MaxWait:            30 * time.Second,
	}
}
