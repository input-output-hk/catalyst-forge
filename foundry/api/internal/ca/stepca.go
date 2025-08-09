package ca

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"encoding/pem"
	"io"
	urlpkg "net/url"

	"github.com/golang-jwt/jwt/v5"
)

// SignRequest represents a minimal CSR sign request payload
type SignRequest struct {
	CSR          string        `json:"csr"`
	RequestedTTL time.Duration `json:"requested_ttl"`
	TemplateData any           `json:"template_data,omitempty"`
}

// SignResponse represents a minimal successful sign response shape
type SignResponse struct {
	Certificate string    `json:"certificate"`
	Chain       []string  `json:"chain"`
	ExpiresAt   time.Time `json:"expires_at"`
	Serial      string    `json:"serial"`
}

// StepCAClient encapsulates access to a step-ca instance with provisioner JWTs
type StepCAClient struct {
	BaseURL     string
	SignPath    string
	Provisioner string
	Issuer      string

	Signer        *ecdsa.PrivateKey
	SignerKID     string
	ProvJWTMaxTTL time.Duration

	HTTP *http.Client
}

// Sign calls step-ca /sign with a short-lived provisioner JWT
func (c *StepCAClient) Sign(ctx context.Context, req SignRequest) (*SignResponse, error) {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	if c.Signer == nil {
		return nil, fmt.Errorf("provisioner signer is nil")
	}
	// Extract subject and keep parsed CSR for SANs
	var subject string
	var parsedCSR *x509.CertificateRequest
	if block, _ := pem.Decode([]byte(req.CSR)); block != nil && block.Type == "CERTIFICATE REQUEST" {
		if parsed, err := x509.ParseCertificateRequest(block.Bytes); err == nil {
			parsedCSR = parsed
			if parsed.Subject.CommonName != "" {
				subject = parsed.Subject.CommonName
			} else if len(parsed.DNSNames) > 0 {
				subject = parsed.DNSNames[0]
			} else if len(parsed.EmailAddresses) > 0 {
				subject = parsed.EmailAddresses[0]
			}
		}
	}
	// Build provisioner JWT
	now := time.Now()
	skew := 5 * time.Second
	// Per step-ca JWK provisioner expectations:
	//   iss: provisioner name
	//   aud: https://<dnsName>/1.0/sign (legacy /sign also accepted); no port
	//   sub: CSR subject (CN or first DNS/email)
	// Build canonical audience from BaseURL host (no port)
	var audiences []string
	if u, err := urlpkg.Parse(c.BaseURL); err == nil {
		host := u.Hostname()
		if host != "" {
			audiences = []string{fmt.Sprintf("https://%s%s", host, c.SignPath)}
		}
	}
	if len(audiences) == 0 {
		audiences = []string{"https://localhost" + c.SignPath}
	}
	// Collect SANs from CSR and include CN if not present
	var sans []string
	if parsedCSR != nil {
		sans = append(sans, parsedCSR.DNSNames...)
		sans = append(sans, parsedCSR.EmailAddresses...)
		for _, ip := range parsedCSR.IPAddresses {
			sans = append(sans, ip.String())
		}
		for _, u := range parsedCSR.URIs {
			if u != nil {
				sans = append(sans, u.String())
			}
		}
	}
	if subject != "" {
		present := false
		for _, s := range sans {
			if s == subject {
				present = true
				break
			}
		}
		if !present {
			sans = append(sans, subject)
		}
	}

	// Use MapClaims so we can include 'sans' alongside standard claims
	claims := jwt.MapClaims{
		"iss":  c.Provisioner,
		"sub":  subject,
		"aud":  audiences, // step-ca accepts string or array; array is fine
		"iat":  now.Add(-skew).Unix(),
		"nbf":  now.Add(-skew).Unix(),
		"exp":  now.Add(c.ProvJWTMaxTTL).Unix(),
		"sans": sans,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	if c.SignerKID != "" {
		token.Header["kid"] = c.SignerKID
	} else {
		// Derive RFC7638 JWK thumbprint KID for the ES256 public key
		// Build minimal JWK with required members in lexicographic order
		pub := c.Signer.Public().(*ecdsa.PublicKey)
		x := pub.X.Bytes()
		y := pub.Y.Bytes()
		// Ensure 32-byte big-endian for P-256 coordinates
		if len(x) < 32 {
			xb := make([]byte, 32)
			copy(xb[32-len(x):], x)
			x = xb
		}
		if len(y) < 32 {
			yb := make([]byte, 32)
			copy(yb[32-len(y):], y)
			y = yb
		}
		jwk := map[string]string{
			"crv": "P-256",
			"kty": "EC",
			"x":   base64.RawURLEncoding.EncodeToString(x),
			"y":   base64.RawURLEncoding.EncodeToString(y),
		}
		b, _ := json.Marshal(jwk)
		sum := sha256.Sum256(b)
		token.Header["kid"] = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	jwtStr, err := token.SignedString(c.Signer)
	if err != nil {
		return nil, fmt.Errorf("failed to sign provisioner jwt: %w", err)
	}

	// Prepare HTTP request with ott and optional notAfter duration
	url := fmt.Sprintf("%s%s", c.BaseURL, c.SignPath)
	body := struct {
		CSR       string `json:"csr"`
		OTT       string `json:"ott"`
		NotAfter  string `json:"notAfter,omitempty"`
		NotBefore string `json:"notBefore,omitempty"`
	}{
		CSR: req.CSR,
		OTT: jwtStr,
	}
	if req.RequestedTTL > 0 {
		body.NotAfter = req.RequestedTTL.String()
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sign request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytesReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("step-ca request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, fmt.Errorf("step-ca error: %s", e.Error)
	}

	// Be tolerant to different step-ca response shapes
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode sign response: %w", err)
	}
	var cert string
	if v, ok := raw["certificate"].(string); ok && v != "" {
		cert = v
	} else if v, ok := raw["crt"].(string); ok && v != "" {
		cert = v
	}
	var chain []string
	if arr, ok := raw["chain"].([]any); ok {
		for _, it := range arr {
			if s, ok := it.(string); ok {
				chain = append(chain, s)
			}
		}
	} else if arr, ok := raw["x509Chain"].([]any); ok {
		for _, it := range arr {
			if s, ok := it.(string); ok {
				chain = append(chain, s)
			}
		}
	}
	if cert == "" {
		return nil, fmt.Errorf("invalid response from step-ca: missing certificate")
	}
	return &SignResponse{Certificate: cert, Chain: chain}, nil
}

// bytesReader avoids importing bytes in tests where not needed
func bytesReader(b []byte) *bytesReaderCompat { return &bytesReaderCompat{b: b} }

type bytesReaderCompat struct {
	b []byte
	i int
}

func (r *bytesReaderCompat) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func (r *bytesReaderCompat) Close() error { return nil }

// Ensure bytesReaderCompat implements io.ReadCloser
var _ io.ReadCloser = (*bytesReaderCompat)(nil)
