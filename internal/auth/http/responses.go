package http

// AuthResponse represents the data returned in response in POST /auth/login.
type AuthResponse struct {
	Token string `json:"token"`
}

// ToLoginResponse converts a JWT token string to an AuthResponse.
func ToLoginResponse(token string) AuthResponse {
	return AuthResponse{Token: token}
}
