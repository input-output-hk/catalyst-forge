package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/certificates"
)

// Client cert test: CSR without DNS/IP SANs
func TestClientCertificateAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	// Generate a CSR with only CN, no SANs
	_, csr, err := generateClientCSR("spiffe://user/test")
	require.NoError(t, err)

	req := &certificates.CertificateSigningRequest{CSR: csr, TTL: "5m"}
	_, err = c.Certificates().SignCertificate(ctx, req)
	require.NoError(t, err)
}

func generateClientCSR(cn string) (*rsa.PrivateKey, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	tpl := x509.CertificateRequest{Subject: pkix.Name{CommonName: cn}}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tpl, key)
	if err != nil {
		return nil, "", err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
	return key, string(pemBytes), nil
}
