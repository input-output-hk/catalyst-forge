package certificates

import "time"

// CertificateSigningRequest represents a request to sign a certificate
type CertificateSigningRequest struct {
	// CSR is the PEM-encoded Certificate Signing Request
	CSR string `json:"csr" binding:"required"`
	
	// SANs are additional Subject Alternative Names to include
	// These will be validated against user permissions
	SANs []string `json:"sans,omitempty"`
	
	// CommonName can override the CN in the CSR
	CommonName string `json:"common_name,omitempty"`
	
	// TTL is the requested certificate lifetime
	// Will be capped by server policy
	TTL string `json:"ttl,omitempty"`
}

// CertificateSigningResponse represents the response after signing a certificate
type CertificateSigningResponse struct {
	// Certificate is the PEM-encoded signed certificate
	Certificate string `json:"certificate"`
	
	// CertificateChain includes intermediate certificates if available
	CertificateChain []string `json:"certificate_chain,omitempty"`
	
	// SerialNumber is the certificate's serial number
	SerialNumber string `json:"serial_number"`
	
	// NotBefore is when the certificate becomes valid
	NotBefore time.Time `json:"not_before"`
	
	// NotAfter is when the certificate expires
	NotAfter time.Time `json:"not_after"`
	
	// Fingerprint is the SHA256 fingerprint of the certificate
	Fingerprint string `json:"fingerprint"`
}

// RootCertificateResponse represents the response for getting the root certificate
type RootCertificateResponse struct {
	// Certificate is the PEM-encoded root certificate
	Certificate []byte `json:"certificate"`
}