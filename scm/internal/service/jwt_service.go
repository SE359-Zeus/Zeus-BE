package service

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID   uuid.UUID
	Role     string
	Email    string
	FullName string
	Status   string
}

type jwtAccessClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Status   string `json:"status"`
	jwt.RegisteredClaims
}

type JWTService struct {
	publicKey *rsa.PublicKey
}

func NewJWTService(pemB64 string) (*JWTService, error) {
	if pemB64 == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY is empty")
	}
	der, err := base64.StdEncoding.DecodeString(pemB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT_PUBLIC_KEY: %w", err)
	}
	block, _ := pem.Decode(der)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from JWT_PUBLIC_KEY")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA public key")
	}
	return &JWTService{publicKey: rsaPub}, nil
}

func (s *JWTService) VerifyAccessToken(tokenString string) (*JWTClaims, error) {
	claims := &jwtAccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrUnauthorized
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, ErrUnauthorized
	}

	parsedClaims, ok := token.Claims.(*jwtAccessClaims)
	if !ok || !token.Valid {
		return nil, ErrUnauthorized
	}

	userID, err := uuid.Parse(parsedClaims.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return &JWTClaims{
		UserID:   userID,
		Role:     parsedClaims.Role,
		Email:    parsedClaims.Email,
		FullName: parsedClaims.FullName,
		Status:   parsedClaims.Status,
	}, nil
}
