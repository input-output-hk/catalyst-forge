package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/certificates"
)

func TestServerCertificateAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	t.Run("SignServerCertificate_ValidCSR", func(t *testing.T) {
		_, csr, err := generateServerCSR("gateway.example.com", []string{"gateway.example.com"})
		require.NoError(t, err)

		req := &certificates.CertificateSigningRequest{
			CSR:        csr,
			CommonName: "gateway.example.com",
			TTL:        "5m",
		}
		resp, err := c.Certificates().SignServerCertificate(ctx, req)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.Certificate)
		block, _ := pem.Decode([]byte(resp.Certificate))
		require.NotNil(t, block)
		cert, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)

		assert.Contains(t, cert.DNSNames, "gateway.example.com")

		// TTL sanity (approximate)
		dur := resp.NotAfter.Sub(resp.NotBefore)
		assert.True(t, dur > 0 && dur <= 24*time.Hour)
	})

	t.Run("SignServerCertificate_Invalid_NoSANs", func(t *testing.T) {
		_, csr, err := generateServerCSRNoSAN("server-no-san")
		require.NoError(t, err)
		_, err = c.Certificates().SignServerCertificate(ctx, &certificates.CertificateSigningRequest{CSR: csr})
		assert.Error(t, err)
	})
}

// generateServerCSR creates a CSR with DNS SANs for server testing
func generateServerCSR(commonName string, dnsNames []string) (*rsa.PrivateKey, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	tpl := x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: commonName},
		DNSNames: dnsNames,
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tpl, key)
	if err != nil {
		return nil, "", err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
	return key, string(pemBytes), nil
}

// generateServerCSRNoSAN creates a CSR without SANs (to trigger validation failure)
func generateServerCSRNoSAN(commonName string) (*rsa.PrivateKey, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	tpl := x509.CertificateRequest{Subject: pkix.Name{CommonName: commonName}}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tpl, key)
	if err != nil {
		return nil, "", err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
	return key, string(pemBytes), nil
}
