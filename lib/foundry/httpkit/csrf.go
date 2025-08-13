package httpkit

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/base64"
    "encoding/binary"
    "errors"
    "fmt"
    "net/http"
    "sync"
    "time"
)

var (
    ErrCSRFTokenMissing  = errors.New("csrf token missing")
    ErrCSRFTokenInvalid  = errors.New("csrf token invalid")
    ErrCSRFHeaderMissing = errors.New("csrf header missing")
)

// CSRF provides CSRF protection for state-changing operations.
type CSRF interface {
    // Generate creates a new CSRF token.
    Generate() (string, error)

    // Validate checks if the request has valid CSRF protection.
    Validate(r *http.Request) error

    // SetCookie sets the CSRF cookie.
    SetCookie(w http.ResponseWriter, token string)

    // ClearCookie removes the CSRF cookie.
    ClearCookie(w http.ResponseWriter)
}

// DefaultCSRF implements header-based CSRF protection.
// IMPORTANT: This is secure ONLY when:
// 1. SameSite=Strict cookies are used (already enforced)
// 2. CORS denies credentialed cross-origin requests
// 3. Custom headers from untrusted origins are not accepted
//
// NOTE: This implementation is header-only and does NOT use cookies for validation.
// The SetCookie method is a no-op to prevent confusion.
type DefaultCSRF struct {
    cookieConfig   CookieConfig
    tokenTTL       time.Duration
    headerName     string
    strictModeOnly bool // If true, validates that security requirements are met
}

// NewDefaultCSRF creates a new CSRF protector with header validation.
// WARNING: Only use this if you have strict CORS and SameSite policies.
// This is a header-only implementation that does NOT validate tokens.
func NewDefaultCSRF(cfg CookieConfig, ttl time.Duration) *DefaultCSRF {
    if cfg.SameSite != http.SameSiteStrictMode {
        panic("DefaultCSRF requires SameSite=Strict for security")
    }
    return &DefaultCSRF{
        cookieConfig:   cfg,
        tokenTTL:       ttl,
        headerName:     "X-Requested-With",
        strictModeOnly: true,
    }
}

// Generate returns a dummy token (not used in validation).
func (c *DefaultCSRF) Generate() (string, error) {
    return "header-only-csrf-protection", nil
}

// Validate checks for valid CSRF protection via custom header.
func (c *DefaultCSRF) Validate(r *http.Request) error {
    header := r.Header.Get(c.headerName)
    if header == "" {
        return ErrCSRFHeaderMissing
    }
    if header != "XMLHttpRequest" {
        return ErrCSRFHeaderMissing
    }
    return nil
}

// SetCookie is a no-op for DefaultCSRF since it's header-only protection.
func (c *DefaultCSRF) SetCookie(w http.ResponseWriter, token string) {}

// ClearCookie is a no-op for DefaultCSRF since it's header-only protection.
func (c *DefaultCSRF) ClearCookie(w http.ResponseWriter) {}

// DoubleSubmitCSRF implements double-submit cookie pattern with token comparison and TTL.
type DoubleSubmitCSRF struct {
    cookieConfig CookieConfig
    tokenTTL     time.Duration
    headerName   string
    secret       []byte // For HMAC-based token validation
}

// NewDoubleSubmitCSRF creates a CSRF protector that compares cookie and header values with TTL enforcement.
func NewDoubleSubmitCSRF(cfg CookieConfig, ttl time.Duration) *DoubleSubmitCSRF {
    secret := make([]byte, 32)
    if _, err := rand.Read(secret); err != nil {
        panic(fmt.Sprintf("failed to generate CSRF secret: %v", err))
    }
    return &DoubleSubmitCSRF{
        cookieConfig: cfg,
        tokenTTL:     ttl,
        headerName:   "X-CSRF-Token",
        secret:       secret,
    }
}

// NewDoubleSubmitCSRFWithSecret creates a CSRF protector with a specific secret.
func NewDoubleSubmitCSRFWithSecret(cfg CookieConfig, ttl time.Duration, secret []byte) *DoubleSubmitCSRF {
    if len(secret) < 32 {
        panic("CSRF secret must be at least 32 bytes")
    }
    return &DoubleSubmitCSRF{
        cookieConfig: cfg,
        tokenTTL:     ttl,
        headerName:   "X-CSRF-Token",
        secret:       secret,
    }
}

// Generate creates a new CSRF token with embedded expiry.
// Token format: base64url(nonce || expiry || hmac(nonce || expiry))
func (c *DoubleSubmitCSRF) Generate() (string, error) {
    nonce := make([]byte, 24)
    if _, err := rand.Read(nonce); err != nil {
        return "", err
    }
    expiry := time.Now().Add(c.tokenTTL).Unix()
    expiryBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(expiryBytes, uint64(expiry))

    tokenData := append(nonce, expiryBytes...)

    h := hmac.New(sha256.New, c.secret)
    h.Write(tokenData)
    mac := h.Sum(nil)

    token := append(tokenData, mac...)
    return base64.URLEncoding.EncodeToString(token), nil
}

// Validate checks that cookie and header tokens match and haven't expired.
func (c *DoubleSubmitCSRF) Validate(r *http.Request) error {
    cookieToken, err := GetCSRFCookie(r)
    if err != nil {
        return ErrCSRFTokenMissing
    }
    headerToken := r.Header.Get(c.headerName)
    if headerToken == "" {
        return ErrCSRFHeaderMissing
    }
    if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
        return ErrCSRFTokenInvalid
    }
    tokenBytes, err := base64.URLEncoding.DecodeString(cookieToken)
    if err != nil {
        return ErrCSRFTokenInvalid
    }
    if len(tokenBytes) != 64 { // 24 + 8 + 32
        return ErrCSRFTokenInvalid
    }
    nonce := tokenBytes[:24]
    expiryBytes := tokenBytes[24:32]
    providedMAC := tokenBytes[32:64]

    tokenData := append(nonce, expiryBytes...)
    h := hmac.New(sha256.New, c.secret)
    h.Write(tokenData)
    expectedMAC := h.Sum(nil)
    if !hmac.Equal(providedMAC, expectedMAC) {
        return ErrCSRFTokenInvalid
    }
    expiry := int64(binary.BigEndian.Uint64(expiryBytes))
    if time.Now().Unix() > expiry {
        return ErrCSRFTokenInvalid
    }
    return nil
}

// SetCookie sets the CSRF cookie.
func (c *DoubleSubmitCSRF) SetCookie(w http.ResponseWriter, token string) {
    SetCSRFCookie(w, token, c.tokenTTL, c.cookieConfig)
}

// ClearCookie removes the CSRF cookie.
func (c *DoubleSubmitCSRF) ClearCookie(w http.ResponseWriter) {
    ClearCSRFCookie(w, c.cookieConfig)
}

// NoOpCSRF provides no CSRF protection (for testing only).
type NoOpCSRF struct{}

func (n *NoOpCSRF) Generate() (string, error)            { return "test-token", nil }
func (n *NoOpCSRF) Validate(r *http.Request) error       { return nil }
func (n *NoOpCSRF) SetCookie(w http.ResponseWriter, _ string) {}
func (n *NoOpCSRF) ClearCookie(w http.ResponseWriter)    {}

// MemoryCSRF stores CSRF tokens in memory with expiration (for testing).
type MemoryCSRF struct {
    mu              sync.RWMutex
    tokens          map[string]time.Time
    cookieConfig    CookieConfig
    tokenTTL        time.Duration
    headerName      string
    lastCleanup     time.Time
    cleanupInterval time.Duration
}

// NewMemoryCSRF creates an in-memory CSRF store for testing.
func NewMemoryCSRF(cfg CookieConfig, ttl time.Duration) *MemoryCSRF {
    return &MemoryCSRF{
        tokens:          make(map[string]time.Time),
        cookieConfig:    cfg,
        tokenTTL:        ttl,
        headerName:      "X-CSRF-Token",
        lastCleanup:     time.Now(),
        cleanupInterval: time.Minute,
    }
}

// Generate creates and stores a new CSRF token.
func (m *MemoryCSRF) Generate() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    token := base64.URLEncoding.EncodeToString(b)
    m.mu.Lock()
    defer m.mu.Unlock()
    m.tokens[token] = time.Now().Add(m.tokenTTL)
    if time.Since(m.lastCleanup) > m.cleanupInterval {
        m.cleanExpiredLocked()
        m.lastCleanup = time.Now()
    }
    return token, nil
}

// Validate checks token validity.
func (m *MemoryCSRF) Validate(r *http.Request) error {
    token := r.Header.Get(m.headerName)
    if token == "" {
        return ErrCSRFHeaderMissing
    }
    m.mu.RLock()
    expiry, exists := m.tokens[token]
    m.mu.RUnlock()
    if !exists {
        return ErrCSRFTokenInvalid
    }
    if time.Now().After(expiry) {
        return ErrCSRFTokenInvalid
    }
    return nil
}

// SetCookie sets the CSRF cookie.
func (m *MemoryCSRF) SetCookie(w http.ResponseWriter, token string) {
    SetCSRFCookie(w, token, m.tokenTTL, m.cookieConfig)
}

// ClearCookie removes the CSRF cookie.
func (m *MemoryCSRF) ClearCookie(w http.ResponseWriter) {
    ClearCSRFCookie(w, m.cookieConfig)
}

// cleanExpiredLocked removes expired tokens from memory (must be called with lock held).
func (m *MemoryCSRF) cleanExpiredLocked() {
    now := time.Now()
    for token, expiry := range m.tokens {
        if now.After(expiry) {
            delete(m.tokens, token)
        }
    }
}

