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
