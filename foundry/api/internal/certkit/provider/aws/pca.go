package aws

import (
	"context"
	"errors"
	"time"
)

// SDKClient is an interface subset of the AWS SDK v2 ACMPCA client.
// This allows easy mocking and decouples from the real SDK here.
type SDKClient interface {
	IssueCertificate(ctx context.Context, in *IssueCertificateInput) (*IssueCertificateOutput, error)
	GetCertificate(ctx context.Context, in *GetCertificateInput) (*GetCertificateOutput, error)
	GetCertificateAuthorityCertificate(ctx context.Context, in *GetCertificateAuthorityCertificateInput) (*GetCertificateAuthorityCertificateOutput, error)
}

type Client struct{ sdk SDKClient }

func New(sdk SDKClient) *Client { return &Client{sdk: sdk} }

// Wire types mirroring what we need without importing the SDK here.
type IssueCertificateInput struct {
	CertificateAuthorityArn *string
	Csr                     []byte
	SigningAlgorithm        string
	TemplateArn             *string
	ValidityNotBefore       *time.Time
	ValidityNotAfter        time.Time
	IdempotencyToken        *string
}
type IssueCertificateOutput struct{ CertificateArn *string }

type GetCertificateInput struct {
	CertificateAuthorityArn *string
	CertificateArn          *string
}
type GetCertificateOutput struct {
	Certificate      *string
	CertificateChain *string
}

type GetCertificateAuthorityCertificateInput struct{ CertificateAuthorityArn *string }
type GetCertificateAuthorityCertificateOutput struct {
	Certificate      *string
	CertificateChain *string
}

func (c *Client) Issue(ctx context.Context, in IssueInput) (string, error) {
	if c.sdk == nil {
		return "", errors.New("pca sdk not configured")
	}
	var tmpl *string
	if in.TemplateArn != "" {
		tmpl = &in.TemplateArn
	}
	var idem *string
	if in.IdempotencyToken != "" {
		idem = &in.IdempotencyToken
	}
	out, err := c.sdk.IssueCertificate(ctx, &IssueCertificateInput{
		CertificateAuthorityArn: &in.CAArn,
		Csr:                     in.CSRDER,
		SigningAlgorithm:        in.SigningAlgorithm,
		TemplateArn:             tmpl,
		ValidityNotBefore:       in.NotBefore,
		ValidityNotAfter:        in.NotAfter,
		IdempotencyToken:        idem,
	})
	if err != nil {
		return "", err
	}
	if out == nil || out.CertificateArn == nil {
		return "", errors.New("empty certificate arn")
	}
	return *out.CertificateArn, nil
}

func (c *Client) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	if c.sdk == nil {
		return "", "", errors.New("pca sdk not configured")
	}
	out, err := c.sdk.GetCertificate(ctx, &GetCertificateInput{CertificateAuthorityArn: &caArn, CertificateArn: &certArn})
	if err != nil {
		return "", "", err
	}
	if out == nil || out.Certificate == nil {
		return "", "", errors.New("certificate not ready")
	}
	cert := *out.Certificate
	chain := ""
	if out.CertificateChain != nil {
		chain = *out.CertificateChain
	}
	return cert, chain, nil
}

func (c *Client) GetCACertificate(ctx context.Context, caArn string) (string, string, error) {
	if c.sdk == nil {
		return "", "", errors.New("pca sdk not configured")
	}
	out, err := c.sdk.GetCertificateAuthorityCertificate(ctx, &GetCertificateAuthorityCertificateInput{CertificateAuthorityArn: &caArn})
	if err != nil {
		return "", "", err
	}
	if out == nil || out.Certificate == nil {
		return "", "", errors.New("empty ca cert")
	}
	cert := *out.Certificate
	chain := ""
	if out.CertificateChain != nil {
		chain = *out.CertificateChain
	}
	return cert, chain, nil
}
