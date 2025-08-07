package certificates

import (
	"context"
	"fmt"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/certificates.go . CertificatesClientInterface

// CertificatesClientInterface defines the interface for certificate operations
type CertificatesClientInterface interface {
	SignCertificate(ctx context.Context, req *CertificateSigningRequest) (*CertificateSigningResponse, error)
	GetRootCertificate(ctx context.Context) ([]byte, error)
}

// CertificatesClient handles certificate-related operations
type CertificatesClient struct {
	do    func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
	doRaw func(ctx context.Context, method, path string, reqBody interface{}) ([]byte, error)
}

// Ensure CertificatesClient implements CertificatesClientInterface
var _ CertificatesClientInterface = (*CertificatesClient)(nil)

// NewCertificatesClient creates a new certificates client
func NewCertificatesClient(
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error,
	doRaw func(ctx context.Context, method, path string, reqBody interface{}) ([]byte, error),
) *CertificatesClient {
	return &CertificatesClient{
		do:    do,
		doRaw: doRaw,
	}
}

// SignCertificate signs a Certificate Signing Request (CSR) using step-ca
func (c *CertificatesClient) SignCertificate(ctx context.Context, req *CertificateSigningRequest) (*CertificateSigningResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	
	if req.CSR == "" {
		return nil, fmt.Errorf("CSR cannot be empty")
	}
	
	var response CertificateSigningResponse
	err := c.do(ctx, "POST", "/certificates/sign", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetRootCertificate retrieves the Certificate Authority's root certificate
func (c *CertificatesClient) GetRootCertificate(ctx context.Context) ([]byte, error) {
	// Use the doRaw function to handle the PEM content type response
	rootCert, err := c.doRaw(ctx, "GET", "/certificates/root", nil)
	if err != nil {
		return nil, err
	}
	return rootCert, nil
}