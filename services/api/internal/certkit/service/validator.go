package service

import (
	"crypto/x509"
)

// ExtractSANs collects SANs from a parsed CSR.
func ExtractSANs(csr *x509.CertificateRequest) (dns []string, uris []string, ips []string) {
	dns = append(dns, csr.DNSNames...)
	for _, u := range csr.URIs {
		uris = append(uris, u.String())
	}
	for _, ip := range csr.IPAddresses {
		ips = append(ips, ip.String())
	}
	return dns, uris, ips
}
