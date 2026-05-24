package service

import (
	"context"
	"log"
	"math"
	"strings"
	"time"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	repo        repository.UserRepository
	rbacSvc     EndpointRBACService
	emailSender EmailService
	cacheRepo   repository.UserCacheRepository
}

func NewUserService(
	repo repository.UserRepository,
	rbacSvc EndpointRBACService,
	emailSender EmailService,
	cacheRepo repository.UserCacheRepository,
) UserService {
	return &userService{
		repo:        repo,
		rbacSvc:     rbacSvc,
		emailSender: emailSender,
		cacheRepo:   cacheRepo,
	}
}

func (s *userService) Create(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	req.Email = strings.TrimSpace(req.Email)
	req.FullName = strings.TrimSpace(req.FullName)

	if req.Email == "" {
		return nil, ErrEmptyEmail
	}
	if !strings.Contains(req.Email, "@") {
		return nil, ErrInvalidEmail
	}
	if req.Password == "" {
		return nil, ErrEmptyPassword
	}
	if len(req.Password) < 8 {
		return nil, ErrShortPassword
	}
	if req.FullName == "" {
		return nil, ErrEmptyName
	}
	if err := s.rbacSvc.ValidateRole(ctx, req.Role); err != nil {
		return nil, ErrInvalidRole
	}

	existing, _ := s.repo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrDuplicateEmail
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Role:         req.Role,
		Status:       models.AccountStatusActive,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	if s.cacheRepo != nil {
		_ = s.cacheRepo.Set(ctx, user)
	}

	if s.emailSender != nil {
		go func(to, fullName, password, role string, createdAt time.Time) {
			bgCtx := context.Background()
			if err := s.emailSender.SendTemplate(bgCtx, EmailTemplateRequest{
				To:       to,
				Subject:  "Your Zeus System account is ready",
				Template: "create_account.html",
				Data: CreateAccountEmailData{
					To:        to,
					FullName:  fullName,
					Username:  to,
					Password:  password,
					Role:      role,
					CreatedAt: createdAt,
				},
			}); err != nil {
				log.Printf("Warning: failed to send create-account email to %s: %v", to, err)
			}
		}(user.Email, user.FullName, req.Password, user.Role, user.CreatedAt)
	}

	return user, nil
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}

	if s.cacheRepo != nil {
		if cached, err := s.cacheRepo.GetByID(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	if s.cacheRepo != nil {
		_ = s.cacheRepo.Set(ctx, user)
	}

	return user, nil
}

func (s *userService) List(ctx context.Context, page, limit int, q string) ([]models.User, *models.PaginationMeta, error) {
	users, totalRows, err := s.repo.List(ctx, page, limit, q)
	if err != nil {
		return nil, nil, err
	}
	if users == nil {
		users = []models.User{}
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 15
	}
	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))
	meta := &models.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalRows:  totalRows,
		TotalPages: totalPages,
	}
	return users, meta, nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	if req.FullName != nil {
		name := strings.TrimSpace(*req.FullName)
		if name == "" {
			return nil, ErrEmptyName
		}
		user.FullName = name
	}
	if req.Role != nil {
		if err := s.rbacSvc.ValidateRole(ctx, *req.Role); err != nil {
			return nil, ErrInvalidRole
		}
		user.Role = *req.Role
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	if s.cacheRepo != nil {
		_ = s.cacheRepo.Set(ctx, user)
	}

	return user, nil
}

func (s *userService) SetStatus(ctx context.Context, id uuid.UUID, status models.AccountStatus) error {
	if id == uuid.Nil {
		return ErrNilID
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}

	if err := s.repo.SetStatus(ctx, id, status); err != nil {
		return err
	}

	updatedUser, err := s.repo.GetByID(ctx, id)
	if err == nil && updatedUser != nil && s.cacheRepo != nil {
		_ = s.cacheRepo.Set(ctx, updatedUser)
	}

	return nil
}

func (s *userService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	email = strings.TrimSpace(email)

	if email == "" {
		return nil, ErrEmptyEmail
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}

	var user *models.User
	var err error
	if s.cacheRepo != nil {
		if cached, err := s.cacheRepo.GetByEmail(ctx, email); err == nil && cached != nil {
			user = cached
		}
	}

	if user == nil {
		user, err = s.repo.GetByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrNotFound
		}
		if s.cacheRepo != nil {
			_ = s.cacheRepo.Set(ctx, user)
		}
	}

	if user.Status == models.AccountStatusInactive {
		return nil, ErrInactiveAccount
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}
