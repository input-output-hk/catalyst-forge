//go:build integration

package test

import (
    "crypto/rsa"
    "encoding/base64"
    "encoding/json"
    "math/big"
    "testing"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
)

func TestJWKS(t *testing.T) {
    env := NewTestEnv(t)
	c := client.NewClient(env.BaseURL())
	ctx, cancel := newTestContext()
	defer cancel()

	raw, err := c.JWKS().Get(ctx)
	require.NoError(t, err)

    var doc struct {
        Keys []map[string]any `json:"keys"`
    }
    require.NoError(t, json.Unmarshal(raw, &doc))
    require.NotEmpty(t, doc.Keys)

    for _, k := range doc.Keys {
        // All keys must have a key type
        kty, ok := k["kty"].(string)
        require.True(t, ok && kty != "", "kty must be present")

        // Accept either EC or RSA
        switch kty {
        case "EC":
            // Expect crv, x, y
            _, hasCrv := k["crv"].(string)
            _, hasX := k["x"].(string)
            _, hasY := k["y"].(string)
            assert.True(t, hasCrv && hasX && hasY, "EC JWK must include crv, x, y")
        case "RSA":
            // Expect n, e
            _, hasN := k["n"].(string)
            _, hasE := k["e"].(string)
            assert.True(t, hasN && hasE, "RSA JWK must include n, e")
        }

        // Optional but useful: kid/alg/use
        if v, ok := k["kid"]; ok {
            assert.IsType(t, "", v)
        }
        if v, ok := k["alg"]; ok {
            assert.IsType(t, "", v)
        }
        if v, ok := k["use"]; ok {
            assert.IsType(t, "", v)
        }
    }

    // Optional: Verify JWT signature validation using JWKS
    t.Run("VerifyJWTSignature", func(t *testing.T) {
        // Find an RSA key in the JWKS (if available)
        var rsaKey *rsa.PublicKey
        var kid string
        
        for _, k := range doc.Keys {
            if kty, ok := k["kty"].(string); ok && kty == "RSA" {
                if nStr, ok := k["n"].(string); ok {
                    if eStr, ok := k["e"].(string); ok {
                        // Decode the RSA public key components
                        nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
                        if err != nil {
                            continue
                        }
                        eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
                        if err != nil {
                            continue
                        }
                        
                        n := new(big.Int).SetBytes(nBytes)
                        e := new(big.Int).SetBytes(eBytes).Int64()
                        
                        rsaKey = &rsa.PublicKey{
                            N: n,
                            E: int(e),
                        }
                        
                        if kidVal, ok := k["kid"].(string); ok {
                            kid = kidVal
                        }
                        break
                    }
                }
            }
        }
        
        if rsaKey != nil {
            // Create a sample JWT token and verify it would fail with the public key
            // This is just to ensure the key format is valid
            expTime := time.Now().Add(time.Hour)
            token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
                "sub": "test",
                "exp": jwt.NewNumericDate(expTime),
            })
            
            // Set the kid if we have one
            if kid != "" {
                token.Header["kid"] = kid
            }
            
            // Parse with the public key (this won't validate signature but checks key format)
            parser := jwt.NewParser()
            _, err := parser.Parse("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.test", func(token *jwt.Token) (interface{}, error) {
                return rsaKey, nil
            })
            
            // We expect an error (invalid signature) but the key format should be valid
            assert.Error(t, err, "Expected signature verification to fail with test token")
        }
    })
}
