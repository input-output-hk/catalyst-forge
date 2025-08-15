package api

import (
	"context"
	"time"

	certkit "github.com/input-output-hk/catalyst-forge/foundry/api/internal/certkit/certkit"
)

// newPKIFake returns a simple fake PCA client for integration tests.
func newPKIFake() certkit.PCAClient {
	return &pcaFake{}
}

type pcaFake struct{ issued bool }

func (p *pcaFake) Issue(_ context.Context, _ certkit.PCAIssueInput) (string, error) {
	p.issued = true
	return "arn:fake:1", nil
}

func (p *pcaFake) GetCertificate(_ context.Context, _ string, _ string) (string, string, error) {
	if !p.issued {
		return "", "", nil
	}
	return "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----\n", "", nil
}

func (p *pcaFake) GetCACertificate(_ context.Context, _ string) (string, string, error) {
	_ = time.Now()
	return "-----BEGIN CERTIFICATE-----\nFAKE-CA\n-----END CERTIFICATE-----\n", "", nil
}
