package ca

import (
	"crypto/x509"
	"errors"
	"net"
)

// ValidateClientCSR enforces strict rules for developer/CI client CSRs.
// Rules:
// - Reject DNS/IP SANs (clients should use URI SANs like spiffe://).
// - CSR must have a valid signature.
func ValidateClientCSR(csr *x509.CertificateRequest) error {
	if csr == nil {
		return errors.New("csr is nil")
	}
	if err := csr.CheckSignature(); err != nil {
		return err
	}
	if len(csr.DNSNames) > 0 {
		return errors.New("client CSR must not include DNS SANs")
	}
	if len(csr.IPAddresses) > 0 {
		return errors.New("client CSR must not include IP SANs")
	}
	return nil
}

// ValidateServerCSR enforces strict rules for server/gateway CSRs.
// Rules:
// - Must include at least one DNS or IP SAN.
// - CSR must have a valid signature.
func ValidateServerCSR(csr *x509.CertificateRequest) error {
	if csr == nil {
		return errors.New("csr is nil")
	}
	if err := csr.CheckSignature(); err != nil {
		return err
	}
	if len(csr.DNSNames) == 0 && len(csr.IPAddresses) == 0 {
		return errors.New("server CSR must include at least one DNS or IP SAN")
	}
	// Optional: validate each IP is a valid address
	for _, ip := range csr.IPAddresses {
		if ip == nil || ip.Equal(net.IP{}) {
			return errors.New("server CSR contains invalid IP SAN")
		}
	}
	return nil
}
