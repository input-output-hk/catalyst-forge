package pca

import (
	"context"
	"time"
)

// PCAClient defines the minimal interface we need from ACM-PCA
type PCAClient interface {
	Issue(ctx context.Context, caArn string, templateArn string, signingAlgorithm string, csrDER []byte, ttl time.Duration, apiPassthroughSANs SANs) (certArn string, err error)
	Get(ctx context.Context, caArn string, certArn string) (certPEM string, chainPEM string, err error)
	// Optional: fetch CA certificate for root endpoint
	GetCA(ctx context.Context, caArn string) (caPEM string, chainPEM string, err error)
}

// SANs captures the APIPassthrough SAN parameters we care about
type SANs struct {
	URIs   []string
	DNS    []string
	Emails []string
	IPs    []string
}
