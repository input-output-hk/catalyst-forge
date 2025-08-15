package certkit

import (
	"context"

	provaws "github.com/input-output-hk/catalyst-forge/services/api/internal/certkit/provider/aws"
)

// awsPCAAdapter adapts provider/aws.Client to certkit.PCAClient.
type awsPCAAdapter struct{ inner *provaws.Client }

func (a awsPCAAdapter) Issue(ctx context.Context, in PCAIssueInput) (string, error) {
	return a.inner.Issue(ctx, provaws.IssueInput{
		CAArn:            in.CAArn,
		CSRDER:           in.CSRDER,
		SigningAlgorithm: in.SigningAlgorithm,
		TemplateArn:      in.TemplateArn,
		NotBefore:        in.NotBefore,
		NotAfter:         in.NotAfter,
		IdempotencyToken: in.IdempotencyToken,
	})
}

func (a awsPCAAdapter) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	return a.inner.GetCertificate(ctx, caArn, certArn)
}

func (a awsPCAAdapter) GetCACertificate(ctx context.Context, caArn string) (string, string, error) {
	return a.inner.GetCACertificate(ctx, caArn)
}
