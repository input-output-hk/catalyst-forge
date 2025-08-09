package pca

import (
	"context"
	"time"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
)

type Options struct {
	Timeout time.Duration
}

type awsPCA struct {
	cli     *acmpca.Client
	timeout time.Duration
}

// NewAWS creates a PCA client backed by AWS SDK v2 using default config chain.
func NewAWS(opts Options) (PCAClient, error) {
	cfg, err := awscfg.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &awsPCA{cli: acmpca.NewFromConfig(cfg), timeout: opts.Timeout}, nil
}

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

func (a *awsPCA) Issue(ctx context.Context, caArn, templateArn, signingAlgorithm string, csrDER []byte, ttl time.Duration, apiPassthroughSANs SANs) (string, error) {
	ictx, cancel := withTimeout(ctx, a.timeout)
	defer cancel()
	// ACM-PCA expects minutes/days/months/years via Validity; use minutes universally
	validity := int64(ttl.Minutes())
	in := &acmpca.IssueCertificateInput{
		CertificateAuthorityArn: &caArn,
		SigningAlgorithm:        types.SigningAlgorithm(signingAlgorithm),
		Validity:                &types.Validity{Type: types.ValidityPeriodType("MINUTES"), Value: &validity},
		TemplateArn:             &templateArn,
		Csr:                     csrDER,
	}
	// For now, omit ApiPassthrough and rely on CSR content per APIPassthrough templates
	out, err := a.cli.IssueCertificate(ictx, in)
	if err != nil {
		return "", err
	}
	return *out.CertificateArn, nil
}

func (a *awsPCA) Get(ctx context.Context, caArn, certArn string) (string, string, error) {
	gctx, cancel := withTimeout(ctx, a.timeout)
	defer cancel()
	out, err := a.cli.GetCertificate(gctx, &acmpca.GetCertificateInput{CertificateAuthorityArn: &caArn, CertificateArn: &certArn})
	if err != nil {
		return "", "", err
	}
	return *out.Certificate, *out.CertificateChain, nil
}

func (a *awsPCA) GetCA(ctx context.Context, caArn string) (string, string, error) {
	gctx, cancel := withTimeout(ctx, a.timeout)
	defer cancel()
	out, err := a.cli.GetCertificateAuthorityCertificate(gctx, &acmpca.GetCertificateAuthorityCertificateInput{CertificateAuthorityArn: &caArn})
	if err != nil {
		return "", "", err
	}
	// Some fields may be nil depending on CA configuration
	var ca, chain string
	if out.Certificate != nil {
		ca = *out.Certificate
	}
	if out.CertificateChain != nil {
		chain = *out.CertificateChain
	}
	return ca, chain, nil
}
