package auth

// ChallengeRequest represents the request body for creating a challenge
type ChallengeRequest struct {
	Email string `json:"email"`
	Kid   string `json:"kid"`
}

// ChallengeResponse represents the response body for a challenge request
type ChallengeResponse struct {
	Token string `json:"token"`
}

// LoginRequest represents the request body for authentication
type LoginRequest struct {
	Token     string `json:"token"`
	Signature string `json:"signature"`
}

// LoginResponse represents the response body for authentication
type LoginResponse struct {
	Token string `json:"token"`
}

// DeviceRegistrationInitRequest represents the request body for initializing device registration
type DeviceRegistrationInitRequest struct {
	InviteID int    `json:"invite_id"`
	Token    string `json:"token"`
}

// DeviceRegistrationInitResponse represents the response body for device registration initialization
type DeviceRegistrationInitResponse struct {
	Alg       string `json:"alg"`
	Challenge string `json:"challenge"`
	DeviceID  string `json:"device_id"`
	ExpiresAt string `json:"expires_at"`
}

// DeviceRegisterRequest represents the request body for completing device registration
type DeviceRegisterRequest struct {
	DeviceID     string                 `json:"device_id"`
	DeviceName   string                 `json:"device_name"`
	DeviceProof  string                 `json:"device_proof"`
	PublicKeyJWK map[string]interface{} `json:"public_key_jwk"`
	Timestamp    int64                  `json:"timestamp"`
}

// DeviceRegisterResponse represents the response body for device registration
type DeviceRegisterResponse struct {
	AccessToken string                 `json:"access_token"`
	ExpiresIn   int64                  `json:"expires_in"`
	TokenType   string                 `json:"token_type"`
	User        map[string]interface{} `json:"user"`
}

// DeviceRefreshRequest represents the request body for refreshing access tokens
type DeviceRefreshRequest struct {
	DeviceID string `json:"device_id"`
}

// DeviceRefreshResponse represents the response body for token refresh
type DeviceRefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// DeviceListResponse represents a device in the user's device list
type DeviceListResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
}
