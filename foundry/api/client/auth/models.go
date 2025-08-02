package auth

// ChallengeRequest represents the request body for creating a challenge
type ChallengeRequest struct {
	Email string `json:"email"`
	Kid   string `json:"kid"`
}

// LoginResponse represents the response body for authentication
type LoginResponse struct {
	Token string `json:"token"`
}
