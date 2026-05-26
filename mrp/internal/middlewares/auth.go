package middlewares

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyRole     = "role"
	ContextKeyEmail    = "email"
	ContextKeyFullName = "full_name"
	ContextKeyStatus   = "status"
	ContextKeyClaims   = "jwt_claims"
)

type Middleware func(http.Handler) http.Handler

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Status   string `json:"status"`
}

type TokenVerifier interface {
	VerifyAccessToken(tokenString string) (*JWTClaims, error)
}

type JWTVerifier struct {
	publicKey *rsa.PublicKey
}

func NewJWTVerifier(publicKey *rsa.PublicKey) *JWTVerifier {
	return &JWTVerifier{publicKey: publicKey}
}

func NewJWTVerifierFromFile(path string) (*JWTVerifier, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY_PATH is empty")
	}

	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWT public key from %s: %w", path, err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from JWT public key file %s", path)
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
		}
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA public key")
	}

	return &JWTVerifier{publicKey: rsaPub}, nil
}

func (v *JWTVerifier) VerifyAccessToken(tokenString string) (*JWTClaims, error) {
	if v == nil || v.publicKey == nil {
		return nil, fmt.Errorf("JWT verifier is not configured")
	}

	claims := &jwtAccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	parsedClaims, ok := token.Claims.(*jwtAccessClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if parsedClaims.Issuer != "zeus-system" {
		return nil, fmt.Errorf("invalid token issuer: %s", parsedClaims.Issuer)
	}

	if _, err := uuid.Parse(parsedClaims.UserID); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &JWTClaims{
		UserID:   parsedClaims.UserID,
		Role:     parsedClaims.Role,
		Email:    parsedClaims.Email,
		FullName: parsedClaims.FullName,
		Status:   parsedClaims.Status,
	}, nil
}

func Authenticate(verifier TokenVerifier) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-KEY")
			if apiKey == "" {
				apiKey = r.Header.Get("X-API-Key")
			}

			expectedKey := os.Getenv("MRP_API_KEY")
			if expectedKey != "" && apiKey == expectedKey {
				ctx := context.WithValue(r.Context(), ContextKeyRole, "admin")
				ctx = context.WithValue(ctx, ContextKeyUserID, "api_key")
				ctx = context.WithValue(ctx, ContextKeyEmail, "api-key@zeus.mrp")
				slog.Info("authentication accepted via API Key",
					slog.String("service", "mrp"),
					slog.String("event", "auth_accepted"),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				slog.Warn("authentication rejected",
					slog.String("service", "mrp"),
					slog.String("event", "auth_rejected"),
					slog.String("reason", "missing_or_invalid_auth_header"),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				writeAuthError(w, http.StatusUnauthorized, "missing_or_invalid_auth_header", "missing or invalid authorization header")
				return
			}

			claims, err := verifier.VerifyAccessToken(parts[1])
			if err != nil {
				slog.Warn("authentication rejected",
					slog.String("service", "mrp"),
					slog.String("event", "auth_rejected"),
					slog.String("reason", "invalid_token"),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("error", err.Error()),
				)
				writeAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid access token")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextKeyFullName, claims.FullName)
			ctx = context.WithValue(ctx, ContextKeyStatus, claims.Status)

			slog.Info("authentication accepted",
				slog.String("service", "mrp"),
				slog.String("event", "auth_accepted"),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("user_id", claims.UserID),
				slog.String("role", claims.Role),
				slog.String("email", claims.Email),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireMethodRoles(methodRoles map[string][]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowedRoles, ok := methodRoles[r.Method]
			if !ok {
				slog.Warn("authorization rejected",
					slog.String("service", "mrp"),
					slog.String("event", "authorization_rejected"),
					slog.String("reason", "method_not_allowed"),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				writeAuthError(w, http.StatusForbidden, "forbidden", "access to this endpoint is not allowed")
				return
			}

			role, _ := r.Context().Value(ContextKeyRole).(string)
			if !roleAllowed(role, allowedRoles) {
				slog.Warn("authorization rejected",
					slog.String("service", "mrp"),
					slog.String("event", "authorization_rejected"),
					slog.String("reason", "insufficient_role"),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("role", role),
					slog.Any("allowed_roles", allowedRoles),
				)
				writeAuthError(w, http.StatusForbidden, "forbidden", "insufficient role for this endpoint")
				return
			}

			slog.Info("authorization accepted",
				slog.String("service", "mrp"),
				slog.String("event", "authorization_accepted"),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("role", role),
			)

			next.ServeHTTP(w, r)
		})
	}
}

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

type jwtAccessClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Status   string `json:"status"`
	jwt.RegisteredClaims
}

func roleAllowed(role string, allowed []string) bool {
	if strings.EqualFold(role, "admin") {
		return true
	}
	for _, candidate := range allowed {
		if strings.EqualFold(role, candidate) {
			return true
		}
	}
	return false
}

type responseEnvelope struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Metadata   any    `json:"metadata"`
	Data       any    `json:"data"`
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responseEnvelope{
		Message:    message,
		StatusCode: status,
		Metadata:   map[string]any{"code": code},
		Data:       nil,
	})
}
