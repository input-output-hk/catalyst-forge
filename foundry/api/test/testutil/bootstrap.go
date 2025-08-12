//go:build integration

package testutil

import (
    "context"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "time"
)

type BootstrapInvite struct {
    ID    uint   `json:"id"`
    Token string `json:"token"`
}

type DeviceInitResponse struct {
    DeviceID  string    `json:"device_id"`
    Challenge string    `json:"challenge"`
    Alg       string    `json:"alg"`
    ExpiresAt time.Time `json:"expires_at"`
}

type DeviceRegisterResponse struct {
    AccessToken string                 `json:"access_token"`
    TokenType   string                 `json:"token_type"`
    ExpiresIn   int                    `json:"expires_in"`
    User        map[string]interface{} `json:"user"`
}


// BootstrapAdmin uses /auth/bootstrap then device registration to get an admin access token.
func BootstrapAdmin(ctx context.Context, baseURL, bootstrapToken, email string) (accessToken string, err error) {
    // 1) Call /auth/bootstrap to create an invite
    var inv BootstrapInvite
    _, err = DoJSON(nil, "POST", baseURL+"/auth/bootstrap", nil, map[string]string{
        "email":           email,
        "bootstrap_token": bootstrapToken,
    }, &inv)
    if err != nil { return "", fmt.Errorf("bootstrap: %w", err) }

    // 2) Device registration init
    var initResp DeviceInitResponse
    _, err = DoJSON(nil, "POST", baseURL+"/auth/devices/init", nil, map[string]any{
        "token":     inv.Token,
        "invite_id": inv.ID,
    }, &initResp)
    if err != nil { return "", fmt.Errorf("init: %w", err) }

    // 3) Generate ECDSA P-256 key
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil { return "", fmt.Errorf("failed to generate ECDSA key: %w", err) }

    // 4) Build JWK
    jwk := map[string]any{
        "kty": "EC",
        "crv": "P-256",
        "x":   base64.RawURLEncoding.EncodeToString(priv.PublicKey.X.Bytes()),
        "y":   base64.RawURLEncoding.EncodeToString(priv.PublicKey.Y.Bytes()),
    }

    // 5) Create DEVICE-REGISTER proof: ts.base64url(ASN.1 DER Sign)
    ts := time.Now().Unix()
    canonical := fmt.Sprintf("DEVICE-REGISTER\n%d\n%s\n%s", ts, initResp.DeviceID, initResp.Challenge)
    sum := sha256.Sum256([]byte(canonical))
    sig, err := ecdsa.SignASN1(rand.Reader, priv, sum[:])
    if err != nil { return "", fmt.Errorf("failed to sign device proof: %w", err) }
    proof := fmt.Sprintf("%d.%s", ts, base64.RawURLEncoding.EncodeToString(sig))

    // 6) Register
    var regResp DeviceRegisterResponse
    _, err = DoJSON(nil, "POST", baseURL+"/auth/devices/register", nil, map[string]any{
        "device_id":      initResp.DeviceID,
        "device_name":    "test-admin-device",
        "public_key_jwk": jwk,
        "device_proof":   proof,
        "timestamp":      ts,
    }, &regResp)
    if err != nil { return "", fmt.Errorf("register: %w", err) }

    return regResp.AccessToken, nil
}

