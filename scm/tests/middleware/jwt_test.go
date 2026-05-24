package middleware_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"zeus-scm-service/internal/handler/middleware"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	return key
}

func TestParseAccessToken_Success(t *testing.T) {
	key := generateTestKey(t)
	userID := uuid.New().String()
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id":   userID,
		"role":      "scm_operator",
		"email":     "operator@zeus.com",
		"full_name": "Operator",
		"status":    "ACTIVE",
		"iss":       "zeus-system",
		"iat":       now.Unix(),
		"exp":       now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(key)
	assert.NoError(t, err)

	parsed, err := middleware.ParseAccessToken(tokenStr, &key.PublicKey)
	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Equal(t, userID, parsed.UserID)
	assert.Equal(t, "scm_operator", parsed.Role)
	assert.Equal(t, "operator@zeus.com", parsed.Email)
	assert.Equal(t, "ACTIVE", parsed.Status)
}

func TestParseAccessToken_InvalidIssuer(t *testing.T) {
	key := generateTestKey(t)
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id": uuid.New().String(),
		"role":    "scm_operator",
		"iss":     "wrong-issuer",
		"iat":     now.Unix(),
		"exp":     now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(key)
	assert.NoError(t, err)

	parsed, err := middleware.ParseAccessToken(tokenStr, &key.PublicKey)
	assert.Error(t, err)
	assert.Nil(t, parsed)
	assert.Contains(t, err.Error(), "invalid token issuer")
}

func TestParseAccessToken_Expired(t *testing.T) {
	key := generateTestKey(t)
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id": uuid.New().String(),
		"role":    "scm_operator",
		"iss":     "zeus-system",
		"iat":     now.Add(-2 * time.Hour).Unix(),
		"exp":     now.Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(key)
	assert.NoError(t, err)

	parsed, err := middleware.ParseAccessToken(tokenStr, &key.PublicKey)
	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseAccessToken_WrongSigningMethod(t *testing.T) {
	key := generateTestKey(t)
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id": uuid.New().String(),
		"role":    "scm_operator",
		"iss":     "zeus-system",
		"iat":     now.Unix(),
		"exp":     now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("secret"))
	assert.NoError(t, err)

	parsed, err := middleware.ParseAccessToken(tokenStr, &key.PublicKey)
	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseAccessToken_InvalidToken(t *testing.T) {
	key := generateTestKey(t)

	parsed, err := middleware.ParseAccessToken("invalid-token-string", &key.PublicKey)
	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseAccessToken_WrongKey(t *testing.T) {
	key := generateTestKey(t)
	wrongKey := generateTestKey(t)
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id": uuid.New().String(),
		"role":    "scm_operator",
		"iss":     "zeus-system",
		"iat":     now.Unix(),
		"exp":     now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(key)
	assert.NoError(t, err)

	parsed, err := middleware.ParseAccessToken(tokenStr, &wrongKey.PublicKey)
	assert.Error(t, err)
	assert.Nil(t, parsed)
}
