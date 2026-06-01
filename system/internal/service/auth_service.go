package service

import (
	"context"
	"crypto/rsa"
	"time"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type jwtAccessClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Status   string `json:"status"`
	jwt.RegisteredClaims
}

type jwtRefreshClaims struct {
	JTI uuid.UUID `json:"jti"`
	SUB uuid.UUID `json:"sub"`
	jwt.RegisteredClaims
}

type authService struct {
	userService UserService
	refreshRepo repository.RefreshTokenRepository
	sessionRepo repository.SessionRepository
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
}

func NewAuthService(
	userSvc UserService,
	refreshRepo repository.RefreshTokenRepository,
	sessionRepo repository.SessionRepository,
	privateKey *rsa.PrivateKey,
) AuthService {
	return &authService{
		userService: userSvc,
		refreshRepo: refreshRepo,
		sessionRepo: sessionRepo,
		privateKey:  privateKey,
		publicKey:   &privateKey.PublicKey,
	}
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthLoginResult, error) {
	user, err := s.userService.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(user.ID, user.Role, user.Email, user.FullName, string(user.Status))
	if err != nil {
		return nil, err
	}

	refreshToken, jti, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	if s.refreshRepo != nil {
		_ = s.refreshRepo.SaveRefreshToken(ctx, jti.String(), user.ID.String())
	}

	if err := s.sessionRepo.Create(ctx, &models.Session{
		UserID:    user.ID,
		UserEmail: user.Email,
		JTI:       jti.String(),
		ExpiresAt: time.Now().Add(models.RefreshTokenDuration),
	}); err != nil {
		return nil, err
	}

	return &models.AuthLoginResult{
		Tokens: &models.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int64(models.AccessTokenDuration.Seconds()),
		},
		User: user,
	}, nil
}

func (s *authService) Refresh(ctx context.Context, req models.RefreshRequest) (*models.AuthLoginResult, error) {
	refreshClaims := &jwtRefreshClaims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, refreshClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrUnauthorized
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, ErrUnauthorized
	}

	claims, ok := token.Claims.(*jwtRefreshClaims)
	if !ok || !token.Valid {
		return nil, ErrUnauthorized
	}

	if _, err := s.validateRefreshToken(ctx, claims); err != nil {
		return nil, err
	}

	user, err := s.userService.GetByID(ctx, claims.SUB)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(user.ID, user.Role, user.Email, user.FullName, string(user.Status))
	if err != nil {
		return nil, err
	}

	newRefreshToken, newJTI, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	if s.refreshRepo != nil {
		_ = s.refreshRepo.SaveRefreshToken(ctx, newJTI.String(), user.ID.String())
	}

	if err := s.sessionRepo.Create(ctx, &models.Session{
		UserID:    user.ID,
		UserEmail: user.Email,
		JTI:       newJTI.String(),
		ExpiresAt: time.Now().Add(models.RefreshTokenDuration),
	}); err != nil {
		return nil, err
	}

	if err := s.sessionRepo.DeleteByJTI(ctx, claims.JTI.String()); err != nil {
		return nil, err
	}

	return &models.AuthLoginResult{
		Tokens: &models.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int64(models.AccessTokenDuration.Seconds()),
		},
		User: user,
	}, nil
}

func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.User, error) {
	return s.userService.ChangePassword(ctx, userID, oldPassword, newPassword)
}

func (s *authService) VerifyAccessToken(tokenString string) (*JWTClaims, error) {
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

	if s.refreshRepo != nil && parsedClaims.ID != "" {
		blacklisted, err := s.refreshRepo.IsAccessTokenBlacklisted(context.Background(), parsedClaims.ID)
		if err == nil && blacklisted {
			return nil, ErrUnauthorized
		}
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

func (s *authService) Logout(ctx context.Context, accessToken string) error {
	claims := &jwtAccessClaims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrUnauthorized
		}
		return s.publicKey, nil
	})
	if err != nil {
		return ErrUnauthorized
	}

	parsedClaims, ok := token.Claims.(*jwtAccessClaims)
	if !ok || !token.Valid {
		return ErrUnauthorized
	}

	if err := s.sessionRepo.DeleteByUserID(ctx, parsedClaims.UserID); err != nil {
		return err
	}

	if s.refreshRepo != nil {
		_ = s.refreshRepo.DeleteUserTokens(ctx, parsedClaims.UserID)
		if parsedClaims.ID != "" {
			remaining := time.Until(parsedClaims.ExpiresAt.Time)
			if remaining > 0 {
				_ = s.refreshRepo.BlacklistAccessToken(ctx, parsedClaims.ID, remaining)
			}
		}
	}

	return nil
}

func (s *authService) validateRefreshToken(ctx context.Context, claims *jwtRefreshClaims) (string, error) {
	if s.refreshRepo != nil {
		userID, err := s.refreshRepo.ValidateRefreshToken(ctx, claims.JTI.String())
		if err == nil && userID != "" {
			if userID != claims.SUB.String() {
				return "", ErrUnauthorized
			}
			return userID, nil
		}
	}

	session, err := s.sessionRepo.GetByJTI(ctx, claims.JTI.String())
	if err != nil || session == nil {
		return "", ErrUnauthorized
	}
	if session.UserID.String() != claims.SUB.String() {
		return "", ErrUnauthorized
	}

	return session.UserID.String(), nil
}

func (s *authService) generateAccessToken(userID uuid.UUID, role, email, fullName, status string) (string, error) {
	now := time.Now()
	claims := &jwtAccessClaims{
		UserID:   userID.String(),
		Role:     role,
		Email:    email,
		FullName: fullName,
		Status:   status,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(models.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "zeus-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *authService) generateRefreshToken(userID uuid.UUID) (string, uuid.UUID, error) {
	jti := uuid.New()

	claims := &jwtRefreshClaims{
		JTI: jti,
		SUB: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(models.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "zeus-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", uuid.Nil, err
	}

	return tokenString, jti, nil
}
