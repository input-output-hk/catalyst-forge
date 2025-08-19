package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateES256KeyPairProperties(t *testing.T) {
	key, err := GenerateES256KeyPair()
	require.NoError(t, err)
	require.NotNil(t, key)
	assert.Equal(t, elliptic.P256(), key.Curve)
	assert.NotZero(t, key.D.BitLen())
	assert.NotZero(t, key.X.BitLen())
	assert.NotZero(t, key.Y.BitLen())
}

func TestEncodeDecodePrivateKeyPEM_SEC1_RoundTrip(t *testing.T) {
	key, err := GenerateES256KeyPair()
	require.NoError(t, err)

	pemBytes, err := EncodePrivateKeyPEM(key)
	require.NoError(t, err)

	dec, err := DecodePrivateKeyPEM(pemBytes)
	require.NoError(t, err)
	assert.Equal(t, elliptic.P256(), dec.Curve)
	assert.Equal(t, key.D, dec.D)
	assert.Equal(t, key.X, dec.X)
	assert.Equal(t, key.Y, dec.Y)
}

func TestEncodeDecodePrivateKeyPEM_PKCS8_RoundTrip(t *testing.T) {
	key, err := GenerateES256KeyPair()
	require.NoError(t, err)

	pemBytes, err := EncodePrivateKeyPEMPKCS8(key)
	require.NoError(t, err)

	dec, err := DecodePrivateKeyPEM(pemBytes)
	require.NoError(t, err)
	assert.Equal(t, elliptic.P256(), dec.Curve)
	assert.Equal(t, key.D, dec.D)
	assert.Equal(t, key.X, dec.X)
	assert.Equal(t, key.Y, dec.Y)
}

func TestEncodePublicKeyPEM_RoundTrip(t *testing.T) {
	key, err := GenerateES256KeyPair()
	require.NoError(t, err)

	pemPub, err := EncodePublicKeyPEM(&key.PublicKey)
	require.NoError(t, err)

	block, _ := pem.Decode(pemPub)
	require.NotNil(t, block)
	require.Equal(t, "PUBLIC KEY", block.Type)

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	pub, ok := parsed.(*ecdsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, key.X, pub.X)
	assert.Equal(t, key.Y, pub.Y)
}

func TestDecodePrivateKeyPEM_InvalidPEM(t *testing.T) {
	_, err := DecodePrivateKeyPEM([]byte("not pem"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse PEM block")
}

func TestDecodePrivateKeyPEM_UnsupportedType_RSA(t *testing.T) {
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("dummy")}
	_, err := DecodePrivateKeyPEM(pem.EncodeToMemory(block))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key type")
}

func TestDecodePrivateKeyPEM_PKCS8_RSA(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs8, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})
	_, err = DecodePrivateKeyPEM(pemBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an ECDSA private key")
}

func TestDecodePrivateKeyPEM_WrongCurve_SEC1(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	_, err = DecodePrivateKeyPEM(pemBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key must use P-256 curve")
}

func TestDecodePrivateKeyPEM_WrongCurve_PKCS8(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	_, err = DecodePrivateKeyPEM(pemBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key must use P-256 curve")
}

func TestGenerateKeyIDAt_Quarters(t *testing.T) {
	cases := []struct {
		when time.Time
		want string
	}{
		{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "2025-q1"},
		{time.Date(2025, 3, 31, 23, 59, 59, 0, time.UTC), "2025-q1"},
		{time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), "2025-q2"},
		{time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC), "2025-q2"},
		{time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), "2025-q3"},
		{time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC), "2025-q3"},
		{time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), "2025-q4"},
		{time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), "2025-q4"},
	}
	for _, c := range cases {
		got := GenerateKeyIDAt(c.when)
		assert.Equal(t, c.want, got)
	}
}

func TestGenerateKeyID_Format(t *testing.T) {
	got := GenerateKeyID()
	re := regexp.MustCompile(`^\d{4}-q[1-4]$`)
	assert.True(t, re.MatchString(got), "unexpected key id format: %s", got)
}
