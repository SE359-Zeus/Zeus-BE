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
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
}

func NewAuthService(
	userSvc UserService,
	refreshRepo repository.RefreshTokenRepository,
	privateKey *rsa.PrivateKey,
) AuthService {
	return &authService{
		userService: userSvc,
		refreshRepo: refreshRepo,
		privateKey:  privateKey,
		publicKey:   &privateKey.PublicKey,
	}
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.TokenPair, error) {
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

	if err := s.refreshRepo.SaveRefreshToken(ctx, jti.String(), user.ID.String()); err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(models.AccessTokenDuration.Seconds()),
	}, nil
}

func (s *authService) Refresh(ctx context.Context, req models.RefreshRequest) (*models.TokenPair, error) {
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

	userID, err := s.refreshRepo.ValidateRefreshToken(ctx, claims.JTI.String())
	if err != nil || userID == "" {
		return nil, ErrUnauthorized
	}

	if userID != claims.SUB.String() {
		return nil, ErrUnauthorized
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

	if err := s.refreshRepo.SaveRefreshToken(ctx, newJTI.String(), user.ID.String()); err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(models.AccessTokenDuration.Seconds()),
	}, nil
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

	if parsedClaims.ID != "" {
		blacklisted, err := s.refreshRepo.IsAccessTokenBlacklisted(context.Background(), parsedClaims.ID)
		if err != nil {
			return nil, ErrUnauthorized
		}
		if blacklisted {
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

	if parsedClaims.ID != "" {
		remaining := time.Until(parsedClaims.ExpiresAt.Time)
		if remaining > 0 {
			if err := s.refreshRepo.BlacklistAccessToken(ctx, parsedClaims.ID, remaining); err != nil {
				return err
			}
		}
	}

	return nil
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
