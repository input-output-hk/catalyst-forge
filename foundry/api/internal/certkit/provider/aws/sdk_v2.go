package aws

import (
	"context"
	"math"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	acmpca "github.com/aws/aws-sdk-go-v2/service/acmpca"
	acmpcatypes "github.com/aws/aws-sdk-go-v2/service/acmpca/types"
)

// v2SDKClient adapts the AWS SDK v2 ACMPCA client to our local SDKClient interface.
type v2SDKClient struct{ cli *acmpca.Client }

// NewDefault constructs a Client using AWS default credential/config chain.
func NewDefault(ctx context.Context, region string) (*Client, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return nil, err
	}
	cli := acmpca.NewFromConfig(cfg)
	return New(v2SDKClient{cli: cli}), nil
}

// NewWithClient allows injecting an already-initialized AWS ACMPCA client.
func NewWithClient(cli *acmpca.Client) *Client { return New(v2SDKClient{cli: cli}) }

func (v v2SDKClient) IssueCertificate(ctx context.Context, in *IssueCertificateInput) (*IssueCertificateOutput, error) {
	// Map validity by computing days until NotAfter (PCA supports DAYS/MONTHS/YEARS; pick DAYS)
	var val acmpcatypes.Validity
	if !in.ValidityNotAfter.IsZero() {
		dur := time.Until(in.ValidityNotAfter)
		if dur < 0 {
			dur = 0
		}
		days := int64(math.Ceil(dur.Hours() / 24.0))
		if days <= 0 {
			days = 1
		}
		val = acmpcatypes.Validity{Type: acmpcatypes.ValidityPeriodTypeDays, Value: awsv2.Int64(days)}
	} else {
		// Fallback to 1 day
		val = acmpcatypes.Validity{Type: acmpcatypes.ValidityPeriodTypeDays, Value: awsv2.Int64(1)}
	}
	var notBefore *acmpcatypes.Validity
	if in.ValidityNotBefore != nil {
		nbDur := time.Until(*in.ValidityNotBefore)
		if nbDur > 0 {
			nbDays := int64(math.Ceil(nbDur.Hours() / 24.0))
			notBefore = &acmpcatypes.Validity{Type: acmpcatypes.ValidityPeriodTypeDays, Value: awsv2.Int64(nbDays)}
		}
	}
	req := &acmpca.IssueCertificateInput{
		CertificateAuthorityArn: in.CertificateAuthorityArn,
		Csr:                     in.Csr,
		SigningAlgorithm:        acmpcatypes.SigningAlgorithm(in.SigningAlgorithm),
		Validity:                &val,
		TemplateArn:             in.TemplateArn,
		IdempotencyToken:        in.IdempotencyToken,
	}
	if notBefore != nil {
		req.ValidityNotBefore = notBefore
	}
	out, err := v.cli.IssueCertificate(ctx, req)
	if err != nil {
		return nil, err
	}
	return &IssueCertificateOutput{CertificateArn: out.CertificateArn}, nil
}

func (v v2SDKClient) GetCertificate(ctx context.Context, in *GetCertificateInput) (*GetCertificateOutput, error) {
	out, err := v.cli.GetCertificate(ctx, &acmpca.GetCertificateInput{CertificateAuthorityArn: in.CertificateAuthorityArn, CertificateArn: in.CertificateArn})
	if err != nil {
		return nil, err
	}
	return &GetCertificateOutput{Certificate: out.Certificate, CertificateChain: out.CertificateChain}, nil
}

func (v v2SDKClient) GetCertificateAuthorityCertificate(ctx context.Context, in *GetCertificateAuthorityCertificateInput) (*GetCertificateAuthorityCertificateOutput, error) {
	out, err := v.cli.GetCertificateAuthorityCertificate(ctx, &acmpca.GetCertificateAuthorityCertificateInput{CertificateAuthorityArn: in.CertificateAuthorityArn})
	if err != nil {
		return nil, err
	}
	return &GetCertificateAuthorityCertificateOutput{Certificate: out.Certificate, CertificateChain: out.CertificateChain}, nil
}
