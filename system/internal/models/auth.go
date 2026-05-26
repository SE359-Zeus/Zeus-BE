package models

import "time"

// TokenPair holds both tokens internally; refresh token is delivered via cookie.
type TokenPair struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
	TokenType    string `json:"-"`
	ExpiresIn    int64  `json:"-"`
}

// AuthLoginResult is the composite return type from the auth service for login/refresh.
// The handler uses Tokens for cookie/header and builds AuthResponse for the body.
type AuthLoginResult struct {
	Tokens *TokenPair
	User   *User
}

// AuthResponse is the JSON body returned for login and refresh endpoints.
// The refresh token is intentionally omitted — it is returned via cookie.
type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserResponse `json:"user"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest is kept for the service layer; the handler accepts refresh tokens from body or cookie.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour

	// RefreshTokenCookieName is the cookie name used to transport the refresh token.
	RefreshTokenCookieName = "refresh_token"
)
