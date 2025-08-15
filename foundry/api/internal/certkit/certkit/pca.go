package certkit

import (
	"context"
	"time"
)

// PCAClient defines the minimal AWS PCA operations needed by certkit.
type PCAClient interface {
	Issue(ctx context.Context, in PCAIssueInput) (certArn string, err error)
	GetCertificate(ctx context.Context, caArn, certArn string) (certPEM, chainPEM string, err error)
	GetCACertificate(ctx context.Context, caArn string) (caCertPEM, chainPEM string, err error)
}

// PCAIssueInput captures the inputs to IssueCertificate.
type PCAIssueInput struct {
	CAArn            string
	CSRDER           []byte
	SigningAlgorithm string
	TemplateArn      string
	NotBefore        *time.Time
	NotAfter         time.Time
	IdempotencyToken string
}
