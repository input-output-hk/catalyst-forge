package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
)

// DeviceProofVerifier handles device proof verification for device-keypair authentication.
type DeviceProofVerifier struct {
	deviceRepo userrepo.DeviceRepository
	config     *config.AuthConfig
}

// NewDeviceProofVerifier creates a new device proof verifier.
func NewDeviceProofVerifier(deviceRepo userrepo.DeviceRepository, config *config.AuthConfig) *DeviceProofVerifier {
	return &DeviceProofVerifier{
		deviceRepo: deviceRepo,
		config:     config,
	}
}

// DeviceProofHeaders represents the parsed device proof headers.
type DeviceProofHeaders struct {
	DeviceID uuid.UUID
	Proof    string
}

// DeviceProofData represents the parsed device proof components.
type DeviceProofData struct {
	Timestamp int64
	Signature []byte
}

// ParseDeviceProofHeaders parses and validates the X-Device-Id and X-Device-Proof headers.
func (v *DeviceProofVerifier) ParseDeviceProofHeaders(deviceIDHeader, proofHeader string) (*DeviceProofHeaders, error) {
	if deviceIDHeader == "" {
		return nil, errors.New("X-Device-Id header is required")
	}
	if proofHeader == "" {
		return nil, errors.New("X-Device-Proof header is required")
	}

	deviceID, err := uuid.Parse(deviceIDHeader)
	if err != nil {
		return nil, fmt.Errorf("invalid X-Device-Id format: %w", err)
	}

	return &DeviceProofHeaders{
		DeviceID: deviceID,
		Proof:    proofHeader,
	}, nil
}

// ParseDeviceProof parses the device proof string (timestamp.base64url(signature)).
func (v *DeviceProofVerifier) ParseDeviceProof(proof string) (*DeviceProofData, error) {
	parts := strings.SplitN(proof, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid device proof format: expected 'timestamp.signature'")
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp in device proof: %w", err)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding in device proof: %w", err)
	}

	return &DeviceProofData{
		Timestamp: timestamp,
		Signature: signature,
	}, nil
}

// GenerateCanonicalString generates the canonical string for device authentication
// Format: AUTH-REFRESH\n<timestamp>\n<device_id>\n<origin>\nPOST /auth/refresh.
func (v *DeviceProofVerifier) GenerateCanonicalString(timestamp int64, deviceID uuid.UUID, origin, method, path string) string {
	return fmt.Sprintf("AUTH-REFRESH\n%d\n%s\n%s\n%s %s",
		timestamp,
		deviceID.String(),
		origin,
		method,
		path,
	)
}

// GenerateCanonicalStringForChallenge generates the canonical string for device registration challenge verification.
func (v *DeviceProofVerifier) GenerateCanonicalStringForChallenge(timestamp int64, deviceID uuid.UUID, challenge string) string {
	return fmt.Sprintf("DEVICE-REGISTER\n%d\n%s\n%s",
		timestamp, deviceID.String(), challenge)
}

// VerifyDeviceProofWithKey verifies a device proof signature using the provided JWK public key.
func (v *DeviceProofVerifier) VerifyDeviceProofWithKey(canonicalString, proof string, publicKeyJWK map[string]interface{}) error {
	// CRITICAL: Validate algorithm to prevent algorithm confusion attacks
	if err := v.validateAlgorithm(publicKeyJWK); err != nil {
		return fmt.Errorf("algorithm validation failed: %w", err)
	}

	// Parse the device proof (timestamp.signature format)
	proofData, err := v.ParseDeviceProof(proof)
	if err != nil {
		return err
	}

	// Parse the JWK and extract the ECDSA public key
	publicKey, err := v.parseECDSAFromJWK(publicKeyJWK)
	if err != nil {
		return fmt.Errorf("failed to parse public key from JWK: %w", err)
	}

	// Hash the canonical string
	hash := sha256.Sum256([]byte(canonicalString))

	// Accept either DER (ASN.1) or raw r||s signatures for compatibility
	if ecdsa.VerifyASN1(publicKey, hash[:], proofData.Signature) {
		return nil
	}
	if len(proofData.Signature) == 64 {
		r := new(big.Int).SetBytes(proofData.Signature[:32])
		s := new(big.Int).SetBytes(proofData.Signature[32:])
		if ecdsa.Verify(publicKey, hash[:], r, s) {
			return nil
		}
	}
	return errors.New("signature verification failed")
}

// VerifyTimestamp checks if the timestamp is within acceptable skew tolerance.
func (v *DeviceProofVerifier) VerifyTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	skewSeconds := int64(v.config.RefreshSkew.Seconds())

	if timestamp > now+skewSeconds {
		return fmt.Errorf("device proof timestamp too far in future: %d vs %d (skew: %ds)", timestamp, now, skewSeconds)
	}

	if timestamp < now-skewSeconds {
		return fmt.Errorf("device proof timestamp too far in past: %d vs %d (skew: %ds)", timestamp, now, skewSeconds)
	}

	return nil
}

// GetDevicePublicKey retrieves and parses the device's public key from the database.
func (v *DeviceProofVerifier) GetDevicePublicKey(deviceID uuid.UUID) (*ecdsa.PublicKey, *dbmodel.Device, error) {
	device, err := v.deviceRepo.GetByID(deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, nil, errors.New("device not found")
	}

	// Check device status
	if device.Status != "active" {
		return nil, device, fmt.Errorf("device is not active: %s", device.Status)
	}

	// Parse JWK from JSONB field
	var jwk map[string]interface{}
	if err := json.Unmarshal(device.PublicJWK, &jwk); err != nil {
		return nil, device, fmt.Errorf("failed to parse device JWK: %w", err)
	}

	// CRITICAL: Validate algorithm even for stored keys (defense in depth)
	if err := v.validateAlgorithm(jwk); err != nil {
		return nil, device, fmt.Errorf("stored device key failed algorithm validation: %w", err)
	}

	// Extract ECDSA public key from JWK
	pubKey, err := v.parseECDSAFromJWK(jwk)
	if err != nil {
		return nil, device, fmt.Errorf("failed to parse ECDSA public key from JWK: %w", err)
	}

	return pubKey, device, nil
}

// validateAlgorithm validates that the JWK uses an allowed algorithm
// CRITICAL: This prevents algorithm confusion attacks including "none" algorithm bypass.
func (v *DeviceProofVerifier) validateAlgorithm(jwk map[string]interface{}) error {
	// Check if alg field is present
	if algValue, hasAlg := jwk["alg"]; hasAlg {
		alg, ok := algValue.(string)
		if !ok {
			return errors.New("JWK 'alg' field must be a string")
		}

		// CRITICAL: Explicitly reject dangerous algorithms
		if strings.ToLower(alg) == "none" {
			return errors.New("algorithm 'none' is explicitly forbidden for security reasons")
		}

		// Only allow ES256 (ECDSA with P-256 and SHA-256)
		if alg != "ES256" {
			return fmt.Errorf("unsupported algorithm '%s': only ES256 is allowed", alg)
		}
	}

	// Also validate based on key type - must be EC with P-256
	kty, ok := jwk["kty"].(string)
	if !ok {
		return errors.New("JWK missing 'kty' field")
	}

	// Reject symmetric keys (oct), RSA, OKP, and any other key types
	if kty != "EC" {
		return fmt.Errorf("unsupported key type '%s': only EC (Elliptic Curve) keys are allowed", kty)
	}

	// For EC keys, ensure it's P-256
	crv, ok := jwk["crv"].(string)
	if !ok {
		return errors.New("EC JWK missing 'crv' field")
	}

	if crv != "P-256" {
		return fmt.Errorf("unsupported curve '%s': only P-256 is allowed", crv)
	}

	return nil
}

// parseECDSAFromJWK parses an ECDSA public key from JWK format.
func (v *DeviceProofVerifier) parseECDSAFromJWK(jwk map[string]interface{}) (*ecdsa.PublicKey, error) {
	// Note: validateAlgorithm should be called before this function
	// Double-check key type for defense in depth
	kty, ok := jwk["kty"].(string)
	if !ok || kty != "EC" {
		return nil, errors.New("JWK key type must be 'EC' for ECDSA")
	}

	crv, ok := jwk["crv"].(string)
	if !ok || crv != "P-256" {
		return nil, errors.New("JWK curve must be 'P-256'")
	}

	// Extract x and y coordinates
	xStr, ok := jwk["x"].(string)
	if !ok {
		return nil, errors.New("JWK missing 'x' coordinate")
	}

	yStr, ok := jwk["y"].(string)
	if !ok {
		return nil, errors.New("JWK missing 'y' coordinate")
	}

	// Decode base64url coordinates
	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode y coordinate: %w", err)
	}

	// Create the public key
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(), // P-256 curve
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	return pubKey, nil
}

// VerifySignature verifies the ECDSA signature against the canonical string.
func (v *DeviceProofVerifier) VerifySignature(pubKey *ecdsa.PublicKey, canonicalString string, signature []byte) error {
	// Hash the canonical string
	hash := sha256.Sum256([]byte(canonicalString))

	// Parse DER-encoded signature (r || s format for ECDSA P-256)
	if len(signature) != 64 {
		return fmt.Errorf("invalid ECDSA signature length: expected 64 bytes, got %d", len(signature))
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	// Verify the signature
	if !ecdsa.Verify(pubKey, hash[:], r, s) {
		return errors.New("signature verification failed")
	}

	return nil
}

// VerifyDeviceProof performs complete device proof verification.
func (v *DeviceProofVerifier) VerifyDeviceProof(deviceIDHeader, proofHeader, origin, method, path string) (*dbmodel.Device, error) {
	// Parse headers
	headers, err := v.ParseDeviceProofHeaders(deviceIDHeader, proofHeader)
	if err != nil {
		return nil, fmt.Errorf("invalid device proof headers: %w", err)
	}

	// Parse proof data
	proofData, err := v.ParseDeviceProof(headers.Proof)
	if err != nil {
		return nil, fmt.Errorf("invalid device proof data: %w", err)
	}

	// Verify timestamp
	if err := v.VerifyTimestamp(proofData.Timestamp); err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	// Get device public key
	pubKey, device, err := v.GetDevicePublicKey(headers.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("device key lookup failed: %w", err)
	}

	// Generate canonical string
	canonicalString := v.GenerateCanonicalString(proofData.Timestamp, headers.DeviceID, origin, method, path)

	// Verify signature
	if err := v.VerifySignature(pubKey, canonicalString, proofData.Signature); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Update device last_used_at timestamp
	if err := v.deviceRepo.UpdateLastUsed(headers.DeviceID, time.Now()); err != nil {
		// Log but don't fail the verification for this non-critical update
		// In production, you might want to log this error
	}

	return device, nil
}
