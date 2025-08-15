package signing

import "time"

// SignerIdentity represents a signer's identity from a certificate
type SignerIdentity struct {
	Issuer  string    // OIDC issuer (e.g., https://token.actions.githubusercontent.com)
	Subject string    // OIDC subject (e.g., repo:org/repo:ref:refs/heads/main)
	Email   string    // Optional email address
	SANs    []string  // Subject Alternative Names
	Time    time.Time // When the signature was created
}

// OIDCIdentity represents expected OIDC identity for verification
type OIDCIdentity struct {
	Issuer  string // Expected OIDC issuer
	Subject string // Expected OIDC subject
}

// VerificationReport contains the results of signature verification
type VerificationReport struct {
	Digest         string           // Digest that was verified
	Signed         bool             // Whether any signatures were found
	Signers        []SignerIdentity // List of signers
	BundleVerified bool             // Whether Rekor/Fulcio bundle was verified
	Errors         []string         // Any errors encountered during verification
}