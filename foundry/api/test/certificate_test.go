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

func TestCertificateAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	t.Run("GetRootCertificate", func(t *testing.T) {
		rootCert, err := c.Certificates().GetRootCertificate(ctx)
		require.NoError(t, err, "Failed to get root certificate")

		// Validate the returned certificate
		assert.NotEmpty(t, rootCert, "Root certificate should not be empty")

		// Parse the PEM-encoded certificate
		block, _ := pem.Decode(rootCert)
		require.NotNil(t, block, "Root certificate should be valid PEM")
		assert.Equal(t, "CERTIFICATE", block.Type, "PEM block should be a certificate")

		// Parse the X.509 certificate
		cert, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err, "Should be able to parse root certificate")

		// Validate root certificate properties
		assert.True(t, cert.IsCA, "Root certificate should be a CA")
		assert.Contains(t, cert.Subject.CommonName, "Foundry", "Root certificate should contain 'Foundry' in CN")

		t.Logf("Root Certificate Subject: %s", cert.Subject.String())
		t.Logf("Root Certificate Issuer: %s", cert.Issuer.String())
		t.Logf("Root Certificate Valid From: %s", cert.NotBefore.String())
		t.Logf("Root Certificate Valid Until: %s", cert.NotAfter.String())
	})

	t.Run("SignCertificate", func(t *testing.T) {
		// Generate a test private key and CSR
		privateKey, csr, err := generateTestCSR("test.example.com", []string{"test.example.com", "api.test.example.com"})
		require.NoError(t, err, "Failed to generate test CSR")

		t.Run("ValidCSR", func(t *testing.T) {
			req := &certificates.CertificateSigningRequest{
				CSR:        csr,
				SANs:       []string{"test.example.com", "api.test.example.com"},
				CommonName: "test.example.com",
				TTL:        "5m",
			}

			response, err := c.Certificates().SignCertificate(ctx, req)
			require.NoError(t, err, "Failed to sign certificate")

			// Validate the response structure
			assert.NotEmpty(t, response.Certificate, "Signed certificate should not be empty")
			assert.NotEmpty(t, response.SerialNumber, "Serial number should not be empty")
			assert.NotEmpty(t, response.Fingerprint, "Fingerprint should not be empty")
			assert.False(t, response.NotBefore.IsZero(), "NotBefore should be set")
			assert.False(t, response.NotAfter.IsZero(), "NotAfter should be set")
			assert.True(t, response.NotAfter.After(response.NotBefore), "NotAfter should be after NotBefore")

			t.Logf("Certificate Serial Number: %s", response.SerialNumber)
			t.Logf("Certificate Fingerprint: %s", response.Fingerprint)
			t.Logf("Certificate Valid From: %s", response.NotBefore.String())
			t.Logf("Certificate Valid Until: %s", response.NotAfter.String())

			// Parse and validate the signed certificate
			block, _ := pem.Decode([]byte(response.Certificate))
			require.NotNil(t, block, "Signed certificate should be valid PEM")
			assert.Equal(t, "CERTIFICATE", block.Type, "PEM block should be a certificate")

			cert, err := x509.ParseCertificate(block.Bytes)
			require.NoError(t, err, "Should be able to parse signed certificate")

			// Validate certificate properties
			assert.Equal(t, "test.example.com", cert.Subject.CommonName, "Certificate CN should match request")
			assert.False(t, cert.IsCA, "Signed certificate should not be a CA")

			// Validate SANs
			expectedSANs := []string{"test.example.com", "api.test.example.com"}
			assert.ElementsMatch(t, expectedSANs, cert.DNSNames, "Certificate SANs should match request")

			// Validate that the certificate was signed by the root CA
			rootCertBytes, err := c.Certificates().GetRootCertificate(ctx)
			require.NoError(t, err, "Failed to get root certificate for validation")

			roots := x509.NewCertPool()
			roots.AppendCertsFromPEM(rootCertBytes)

			inters := x509.NewCertPool()
			for _, pem := range response.CertificateChain {
				inters.AppendCertsFromPEM([]byte(pem))
			}

			_, err = cert.Verify(x509.VerifyOptions{
				Roots:         roots,
				Intermediates: inters,
			})
			require.NoError(t, err, "Certificate should be verifiable against root CA")
			t.Logf("Certificate verification successful")

			// Validate the certificate can be used with the private key
			err = validateCertificateKeyPair(cert, privateKey)
			require.NoError(t, err, "Certificate should match the private key")

			t.Logf("Certificate-key pair validation successful")
		})

		t.Run("CustomTTL", func(t *testing.T) {
			req := &certificates.CertificateSigningRequest{
				CSR: csr,
				TTL: "8m", // 8 minutes - within 10m limit
			}

			response, err := c.Certificates().SignCertificate(ctx, req)
			require.NoError(t, err, "Failed to sign certificate with custom TTL")

			// Check that the certificate has the requested lifetime (approximately)
			duration := response.NotAfter.Sub(response.NotBefore)
			expectedDuration := 8 * time.Minute

			// Allow some tolerance (±2 minutes) for processing time and step-ca policy
			tolerance := 2 * time.Minute
			assert.True(t,
				duration >= expectedDuration-tolerance && duration <= expectedDuration+tolerance,
				"Certificate duration should be approximately %v, got %v", expectedDuration, duration)

			t.Logf("Custom TTL test - requested: 8m, actual: %v", duration)
		})
	})

	t.Run("ErrorCases", func(t *testing.T) {
		t.Run("InvalidCSR", func(t *testing.T) {
			req := &certificates.CertificateSigningRequest{
				CSR: "invalid-csr-data",
			}

			_, err := c.Certificates().SignCertificate(ctx, req)
			assert.Error(t, err, "Should fail with invalid CSR")
			t.Logf("Invalid CSR error (expected): %v", err)
		})

		t.Run("EmptyCSR", func(t *testing.T) {
			req := &certificates.CertificateSigningRequest{
				CSR: "",
			}

			_, err := c.Certificates().SignCertificate(ctx, req)
			assert.Error(t, err, "Should fail with empty CSR")
			t.Logf("Empty CSR error (expected): %v", err)
		})

		t.Run("NilRequest", func(t *testing.T) {
			_, err := c.Certificates().SignCertificate(ctx, nil)
			assert.Error(t, err, "Should fail with nil request")
			t.Logf("Nil request error (expected): %v", err)
		})

		t.Run("ExcessiveTTL", func(t *testing.T) {
			_, csr, err := generateTestCSR("long-lived.example.com", []string{"long-lived.example.com"})
			require.NoError(t, err, "Failed to generate test CSR")

			req := &certificates.CertificateSigningRequest{
				CSR: csr,
				TTL: "8760h", // 1 year - should be limited by step-ca policy
			}

			response, err := c.Certificates().SignCertificate(ctx, req)
			if err == nil {
				// If it succeeds, the TTL should be capped
				duration := response.NotAfter.Sub(response.NotBefore)
				maxAllowed := 10 * time.Minute // step-ca limits to 10m

				assert.True(t, duration <= maxAllowed,
					"Certificate duration should be capped to %v, got %v", maxAllowed, duration)

				t.Logf("Excessive TTL was capped to: %v", duration)
			} else {
				// If it fails, that's also acceptable behavior
				t.Logf("Excessive TTL rejected (acceptable): %v", err)
			}
		})
	})

	t.Run("CertificateChain", func(t *testing.T) {
		_, csr, err := generateTestCSR("chain-test.example.com", []string{"chain-test.example.com"})
		require.NoError(t, err, "Failed to generate test CSR")

		req := &certificates.CertificateSigningRequest{
			CSR: csr,
			TTL: "6m", // Ensure we stay within 10m limit
		}

		response, err := c.Certificates().SignCertificate(ctx, req)
		require.NoError(t, err, "Failed to sign certificate")

		// If intermediate certificates are present, validate them
		if len(response.CertificateChain) > 0 {
			t.Logf("Certificate chain contains %d intermediate certificates", len(response.CertificateChain))

			for i, intermediatePEM := range response.CertificateChain {
				block, _ := pem.Decode([]byte(intermediatePEM))
				require.NotNil(t, block, "Intermediate certificate %d should be valid PEM", i)

				cert, err := x509.ParseCertificate(block.Bytes)
				require.NoError(t, err, "Should be able to parse intermediate certificate %d", i)

				assert.True(t, cert.IsCA, "Intermediate certificate %d should be a CA", i)
				t.Logf("Intermediate %d Subject: %s", i, cert.Subject.String())
			}
		} else {
			t.Log("No intermediate certificates in chain (direct signing)")
		}
	})
}

// generateTestCSR creates a private key and CSR for testing
func generateTestCSR(commonName string, dnsNames []string) (*rsa.PrivateKey, string, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}

	// Create certificate request template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Test Organization"},
			Country:      []string{"US"},
		},
		DNSNames: dnsNames,
	}

	// Create CSR
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, "", err
	}

	// Encode as PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})

	return privateKey, string(csrPEM), nil
}

// validateCertificateKeyPair ensures the certificate corresponds to the private key
func validateCertificateKeyPair(cert *x509.Certificate, privateKey *rsa.PrivateKey) error {
	// Extract public key from certificate
	certPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return assert.AnError // Certificate doesn't contain RSA public key
	}

	// Compare public keys
	if certPubKey.N.Cmp(privateKey.PublicKey.N) != 0 || certPubKey.E != privateKey.PublicKey.E {
		return assert.AnError // Public keys don't match
	}

	return nil
}
