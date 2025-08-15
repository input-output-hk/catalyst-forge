package aws

import "time"

// PCAClient defines the minimal AWS PCA operations needed by certkit.
type PCAClient interface {
	Issue(ctx Context, in IssueInput) (certArn string, err error)
	GetCertificate(ctx Context, caArn, certArn string) (certPEM, chainPEM string, err error)
	GetCACertificate(ctx Context, caArn string) (caCertPEM, chainPEM string, err error)
}

// IssueInput captures the inputs to IssueCertificate.
type IssueInput struct {
	CAArn            string
	CSRDER           []byte
	SigningAlgorithm string
	TemplateArn      string
	NotBefore        *time.Time
	NotAfter         time.Time
	IdempotencyToken string
}

// Context is a tiny shim to avoid importing std context in interface surface here.
// Use: type Context = context.Context in real code.
type Context interface{ Done() <-chan struct{} }
