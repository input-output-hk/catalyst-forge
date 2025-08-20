package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"time"
)

type Clock interface{ Now() time.Time }

type PCAClient interface {
	Issue(ctx context.Context, in IssueInput) (string, error)
	GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error)
}

type IssueInput struct {
	CAArn            string
	CSRDER           []byte
	SigningAlgorithm string
	TemplateArn      string
	NotBefore        *time.Time
	NotAfter         time.Time
	IdempotencyToken string
}

type SignRequest struct {
	CSRPEM           []byte
	TemplateKey      string
	SigningAlgorithm string
	TTL              time.Duration
	IdempotencyKey   string
	Requestor        string
}

type SignResult struct {
	CertificatePEM string
	ChainPEM       string
	CertificateArn string
}

type Issuer struct {
	CAArn            string
	PCA              PCAClient
	Clock            Clock
	AllowedTemplates map[string]string
	MaxTTL           time.Duration
	PollInterval     time.Duration
	MaxWait          time.Duration
}

func (i *Issuer) SignCSR(ctx context.Context, req SignRequest) (*SignResult, error) {
	// Parse CSR
	block, _ := pem.Decode(req.CSRPEM)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, ErrCSRInvalid
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, ErrCSRSignature
	}
	// TTL clamp
	now := i.Clock.Now()
	ttl := req.TTL
	if ttl <= 0 {
		ttl = i.MaxTTL
	}
	if ttl > i.MaxTTL {
		ttl = i.MaxTTL
	}
	// Template
	tmpl := i.AllowedTemplates[req.TemplateKey]
	// Issue
	arn, err := i.PCA.Issue(ctx, IssueInput{CAArn: i.CAArn, CSRDER: csr.Raw, SigningAlgorithm: req.SigningAlgorithm, TemplateArn: tmpl, NotAfter: now.Add(ttl), IdempotencyToken: req.IdempotencyKey})
	if err != nil {
		return nil, err
	}
	// Poll until issued or timeout
	pollEvery := i.PollInterval
	if pollEvery <= 0 {
		pollEvery = 500 * time.Millisecond
	}
	deadline := now.Add(i.MaxWait)
	if i.MaxWait <= 0 {
		deadline = now.Add(30 * time.Second)
	}
	for {
		cert, chain, err := i.PCA.GetCertificate(ctx, i.CAArn, arn)
		if err == nil && cert != "" {
			return &SignResult{CertificatePEM: cert, ChainPEM: chain, CertificateArn: arn}, nil
		}
		if i.Clock.Now().After(deadline) {
			return nil, ErrIssuanceTimeout
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollEvery):
		}
	}
}
