package certkit

import (
	"context"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/certkit/service"
)

// pcaAdapter bridges certkit.PCAClient to service.PCAClient by converting inputs.
type pcaAdapter struct{ inner PCAClient }

func (a pcaAdapter) Issue(ctx context.Context, in service.IssueInput) (string, error) {
	return a.inner.Issue(ctx, PCAIssueInput{
		CAArn:            in.CAArn,
		CSRDER:           in.CSRDER,
		SigningAlgorithm: in.SigningAlgorithm,
		TemplateArn:      in.TemplateArn,
		NotBefore:        in.NotBefore,
		NotAfter:         in.NotAfter,
		IdempotencyToken: in.IdempotencyToken,
	})
}

func (a pcaAdapter) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	return a.inner.GetCertificate(ctx, caArn, certArn)
}
