package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"testing"
)

// helper to create a CSR with provided fields and sign it
func makeCSR(t *testing.T, subj pkix.Name, dns []string, ips []net.IP) *x509.CertificateRequest {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key gen: %v", err)
	}
	tpl := &x509.CertificateRequest{
		Subject:            subj,
		DNSNames:           dns,
		IPAddresses:        ips,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	if err != nil {
		t.Fatalf("csr create: %v", err)
	}
	csr, err := x509.ParseCertificateRequest(der)
	if err != nil {
		t.Fatalf("csr parse: %v", err)
	}
	return csr
}

func TestValidateClientCSR_Success_NoDNSNoIP(t *testing.T) {
	csr := makeCSR(t, pkix.Name{CommonName: "client"}, nil, nil)
	if err := ValidateClientCSR(csr); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestValidateClientCSR_Fails_WithDNS(t *testing.T) {
	csr := makeCSR(t, pkix.Name{CommonName: "client"}, []string{"example.com"}, nil)
	if err := ValidateClientCSR(csr); err == nil {
		t.Fatalf("expected error for DNS SAN in client CSR")
	}
}

func TestValidateClientCSR_Fails_WithIP(t *testing.T) {
	csr := makeCSR(t, pkix.Name{CommonName: "client"}, nil, []net.IP{net.ParseIP("192.0.2.10")})
	if err := ValidateClientCSR(csr); err == nil {
		t.Fatalf("expected error for IP SAN in client CSR")
	}
}

func TestValidateServerCSR_Success_WithDNS(t *testing.T) {
	csr := makeCSR(t, pkix.Name{CommonName: "server"}, []string{"api.example.com"}, nil)
	if err := ValidateServerCSR(csr); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestValidateServerCSR_Success_WithIP(t *testing.T) {
	csr := makeCSR(t, pkix.Name{CommonName: "server"}, nil, []net.IP{net.ParseIP("203.0.113.5")})
	if err := ValidateServerCSR(csr); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestValidateServerCSR_Fails_NoSANs(t *testing.T) {
	csr := makeCSR(t, pkix.Name{CommonName: "server"}, nil, nil)
	if err := ValidateServerCSR(csr); err == nil {
		t.Fatalf("expected error when no DNS or IP SANs present")
	}
}
