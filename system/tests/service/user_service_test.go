package service_test

import (
	"context"
	"testing"
	"time"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

var anyCtx = mock.Anything

func setupUserSvc() (service.UserService, *service.MockUserRepository, *service.MockEndpointRBACService) {
	repo := new(service.MockUserRepository)
	rbacSvc := new(service.MockEndpointRBACService)
	svc := service.NewUserService(repo, rbacSvc, nil, nil)
	return svc, repo, rbacSvc
}

func setupUserSvcWithEmail() (service.UserService, *service.MockUserRepository, *service.MockEndpointRBACService, *service.MockEmailService) {
	repo := new(service.MockUserRepository)
	rbacSvc := new(service.MockEndpointRBACService)
	emailSender := new(service.MockEmailService)
	svc := service.NewUserService(repo, rbacSvc, emailSender, nil)
	return svc, repo, rbacSvc, emailSender
}

func validCreateReq() models.CreateUserRequest {
	return models.CreateUserRequest{
		Email:    "admin@zeus.com",
		Password: "securepass123",
		FullName: "Admin User",
		Role:     "admin",
	}
}

func TestUserService_Create_Success(t *testing.T) {
	svc, repo, rbacSvc := setupUserSvc()
	req := validCreateReq()

	rbacSvc.On("ValidateRole", anyCtx, "admin").Return(nil)
	repo.On("GetByEmail", anyCtx, req.Email).Return(nil, nil)
	repo.On("Create", anyCtx, mock.AnythingOfType("*models.User")).Return(nil)

	user, err := svc.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.FullName, user.FullName)
	assert.Equal(t, req.Role, user.Role)
	assert.Equal(t, models.AccountStatusActive, user.Status)
	assert.NotEmpty(t, user.PasswordHash)

	// Verify the hash is of the first 10 chars of UUID
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(user.ID.String()[:10]))
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	rbacSvc.AssertExpectations(t)
}

func TestUserService_Create_SendsAccountEmail(t *testing.T) {
	svc, repo, rbacSvc, emailSender := setupUserSvcWithEmail()
	req := validCreateReq()

	var createdUser *models.User
	rbacSvc.On("ValidateRole", anyCtx, "admin").Return(nil)
	repo.On("GetByEmail", anyCtx, req.Email).Return(nil, nil)
	repo.On("Create", anyCtx, mock.AnythingOfType("*models.User")).Run(func(args mock.Arguments) {
		createdUser = args.Get(1).(*models.User)
	}).Return(nil)
	emailSender.On("SendTemplate", anyCtx, mock.MatchedBy(func(data service.EmailTemplateRequest) bool {
		payload, ok := data.Data.(service.CreateAccountEmailData)
		if !ok || createdUser == nil {
			return false
		}
		expectedPwd := createdUser.ID.String()[:10]
		return data.To == req.Email && data.Template == "create_account.html" && data.Subject == "Your Zeus System account is ready" && payload.To == req.Email && payload.Username == req.Email && payload.Password == expectedPwd && payload.FullName == req.FullName && payload.Role == req.Role
	})).Return(nil)

	user, err := svc.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
	rbacSvc.AssertExpectations(t)
	emailSender.AssertExpectations(t)
}

func TestUserService_Create_RejectsEmptyEmail(t *testing.T) {
	svc, _, _ := setupUserSvc()
	req := validCreateReq()
	req.Email = ""

	user, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, service.ErrEmptyEmail)
	assert.Nil(t, user)
}

func TestUserService_Create_RejectsInvalidEmail(t *testing.T) {
	svc, _, _ := setupUserSvc()
	req := validCreateReq()
	req.Email = "notanemail"

	user, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, service.ErrInvalidEmail)
	assert.Nil(t, user)
}

func TestUserService_Create_RejectsInvalidRole(t *testing.T) {
	svc, _, rbacSvc := setupUserSvc()
	req := validCreateReq()
	req.Role = "SuperAdmin"

	rbacSvc.On("ValidateRole", anyCtx, "SuperAdmin").Return(service.ErrInvalidRole)

	user, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, service.ErrInvalidRole)
	assert.Nil(t, user)
	rbacSvc.AssertExpectations(t)
}

func TestUserService_Create_RejectsEmptyName(t *testing.T) {
	svc, _, _ := setupUserSvc()
	req := validCreateReq()
	req.FullName = ""

	user, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, service.ErrEmptyName)
	assert.Nil(t, user)
}

func TestUserService_Create_RejectsDuplicateEmail(t *testing.T) {
	svc, repo, rbacSvc := setupUserSvc()
	req := validCreateReq()

	rbacSvc.On("ValidateRole", anyCtx, "admin").Return(nil)
	existing := &models.User{Email: req.Email}
	repo.On("GetByEmail", anyCtx, req.Email).Return(existing, nil)

	user, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, service.ErrDuplicateEmail)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
	rbacSvc.AssertExpectations(t)
}

func TestUserService_GetByID_Success(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()
	expected := &models.User{ID: id, Email: "user@zeus.com"}

	repo.On("GetByID", anyCtx, id).Return(expected, nil)

	user, err := svc.GetByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, expected, user)
	repo.AssertExpectations(t)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()

	repo.On("GetByID", anyCtx, id).Return(nil, nil)

	user, err := svc.GetByID(context.Background(), id)
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestUserService_GetByID_RejectsNilUUID(t *testing.T) {
	svc, _, _ := setupUserSvc()

	user, err := svc.GetByID(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, service.ErrNilID)
	assert.Nil(t, user)
}

func TestUserService_List_Success(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	expected := []models.User{{Email: "a@z.com"}, {Email: "b@z.com"}}

	repo.On("List", anyCtx, 1, 15, "").Return(expected, int64(2), nil)

	users, meta, err := svc.List(context.Background(), 1, 15, "")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, int64(2), meta.TotalRows)
	assert.Equal(t, 1, meta.TotalPages)
	repo.AssertExpectations(t)
}

func TestUserService_List_ReturnsEmptySliceNotNil(t *testing.T) {
	svc, repo, _ := setupUserSvc()

	repo.On("List", anyCtx, 1, 15, "").Return(nil, int64(0), nil)

	users, meta, err := svc.List(context.Background(), 1, 15, "")
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Len(t, users, 0)
	assert.Equal(t, int64(0), meta.TotalRows)
	repo.AssertExpectations(t)
}

func TestUserService_List_Search(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	expected := []models.User{{Email: "admin@zeus.com"}}

	repo.On("List", anyCtx, 1, 15, "admin").Return(expected, int64(1), nil)

	users, meta, err := svc.List(context.Background(), 1, 15, "admin")
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, int64(1), meta.TotalRows)
	repo.AssertExpectations(t)
}

func TestUserService_Update_Success(t *testing.T) {
	svc, repo, rbacSvc := setupUserSvc()
	id := uuid.New()
	existing := &models.User{ID: id, FullName: "Old Name", Role: "sales_worker"}
	newName := "New Name"
	newRole := "scm_operator"
	req := models.UpdateUserRequest{FullName: &newName, Role: &newRole}

	rbacSvc.On("ValidateRole", anyCtx, "scm_operator").Return(nil)
	repo.On("GetByID", anyCtx, id).Return(existing, nil)
	repo.On("Update", anyCtx, mock.AnythingOfType("*models.User")).Return(nil)

	user, err := svc.Update(context.Background(), id, req)
	assert.NoError(t, err)
	assert.Equal(t, newName, user.FullName)
	assert.Equal(t, newRole, user.Role)
	repo.AssertExpectations(t)
	rbacSvc.AssertExpectations(t)
}

func TestUserService_Update_NotFound(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()

	repo.On("GetByID", anyCtx, id).Return(nil, nil)

	name := "Name"
	user, err := svc.Update(context.Background(), id, models.UpdateUserRequest{FullName: &name})
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestUserService_Update_RejectsNilID(t *testing.T) {
	svc, _, _ := setupUserSvc()
	name := "Name"

	user, err := svc.Update(context.Background(), uuid.Nil, models.UpdateUserRequest{FullName: &name})
	assert.ErrorIs(t, err, service.ErrNilID)
	assert.Nil(t, user)
}

func TestUserService_Update_RejectsEmptyName(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()
	empty := ""

	repo.On("GetByID", anyCtx, id).Return(&models.User{ID: id}, nil)

	user, err := svc.Update(context.Background(), id, models.UpdateUserRequest{FullName: &empty})
	assert.ErrorIs(t, err, service.ErrEmptyName)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestUserService_Update_RejectsInvalidRole(t *testing.T) {
	svc, repo, rbacSvc := setupUserSvc()
	id := uuid.New()
	badRole := "SuperAdmin"

	rbacSvc.On("ValidateRole", anyCtx, "SuperAdmin").Return(service.ErrInvalidRole)
	repo.On("GetByID", anyCtx, id).Return(&models.User{ID: id}, nil)

	user, err := svc.Update(context.Background(), id, models.UpdateUserRequest{Role: &badRole})
	assert.ErrorIs(t, err, service.ErrInvalidRole)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
	rbacSvc.AssertExpectations(t)
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()
	oldPassword := "oldpass123"
	newPassword := "newpass123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	existing := &models.User{ID: id, Email: "user@zeus.com", PasswordHash: string(hash), Status: models.AccountStatusActive}
	originalHash := existing.PasswordHash

	repo.On("GetByID", anyCtx, id).Return(existing, nil)
	repo.On("Update", anyCtx, mock.MatchedBy(func(user *models.User) bool {
		return user.ID == id && user.Email == existing.Email && user.PasswordHash != originalHash
	})).Return(nil)

	user, err := svc.ChangePassword(context.Background(), id, oldPassword, newPassword)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, id, user.ID)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword))
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass123"), bcrypt.DefaultCost)
	repo.On("GetByID", anyCtx, id).Return(&models.User{ID: id, PasswordHash: string(hash)}, nil)

	user, err := svc.ChangePassword(context.Background(), id, "wrongold", "newpass123")
	assert.ErrorIs(t, err, service.ErrUnauthorized)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestUserService_ChangePassword_RejectsShortNewPassword(t *testing.T) {
	svc, _, _ := setupUserSvc()

	user, err := svc.ChangePassword(context.Background(), uuid.New(), "oldpass123", "short")
	assert.ErrorIs(t, err, service.ErrShortPassword)
	assert.Nil(t, user)
}

func TestUserService_ChangePassword_NotFound(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()
	repo.On("GetByID", anyCtx, id).Return(nil, nil)

	user, err := svc.ChangePassword(context.Background(), id, "oldpass123", "newpass123")
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestUserService_SetStatus_Success(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()

	repo.On("GetByID", anyCtx, id).Return(&models.User{ID: id, Status: models.AccountStatusActive}, nil)
	repo.On("SetStatus", anyCtx, id, models.AccountStatusInactive).Return(nil)

	err := svc.SetStatus(context.Background(), id, models.AccountStatusInactive)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserService_SetStatus_NotFound(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	id := uuid.New()

	repo.On("GetByID", anyCtx, id).Return(nil, nil)

	err := svc.SetStatus(context.Background(), id, models.AccountStatusActive)
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestUserService_SetStatus_RejectsNilID(t *testing.T) {
	svc, _, _ := setupUserSvc()

	err := svc.SetStatus(context.Background(), uuid.Nil, models.AccountStatusActive)
	assert.ErrorIs(t, err, service.ErrNilID)
}

func TestUserService_Authenticate_Success(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	email := "admin@zeus.com"
	password := "securepass123"

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &models.User{
		Email:        email,
		FullName:     "Admin",
		Role:         "admin",
		Status:       models.AccountStatusActive,
		PasswordHash: string(hash),
	}

	repo.On("GetByEmail", anyCtx, email).Return(user, nil)

	result, err := svc.Authenticate(context.Background(), email, password)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, email, result.Email)
}

func TestUserService_Authenticate_WrongPassword(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	email := "admin@zeus.com"
	password := "wrongpass"

	hash, _ := bcrypt.GenerateFromPassword([]byte("securepass123"), bcrypt.DefaultCost)
	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Status:       models.AccountStatusActive,
	}

	repo.On("GetByEmail", anyCtx, email).Return(user, nil)

	result, err := svc.Authenticate(context.Background(), email, password)
	assert.ErrorIs(t, err, service.ErrUnauthorized)
	assert.Nil(t, result)
}

func TestUserService_Authenticate_InactiveAccount(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	email := "inactive@zeus.com"
	user := &models.User{
		Email:  email,
		Status: models.AccountStatusInactive,
	}

	repo.On("GetByEmail", anyCtx, email).Return(user, nil)

	result, err := svc.Authenticate(context.Background(), email, "anypass")
	assert.ErrorIs(t, err, service.ErrInactiveAccount)
	assert.Nil(t, result)
}

func TestUserService_Authenticate_UserNotFound(t *testing.T) {
	svc, repo, _ := setupUserSvc()
	repo.On("GetByEmail", anyCtx, "missing@zeus.com").Return(nil, nil)

	result, err := svc.Authenticate(context.Background(), "missing@zeus.com", "anypass")
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, result)
}

func TestUserService_Authenticate_RejectsEmptyEmail(t *testing.T) {
	svc, _, _ := setupUserSvc()

	result, err := svc.Authenticate(context.Background(), "", "anypass")
	assert.ErrorIs(t, err, service.ErrEmptyEmail)
	assert.Nil(t, result)
}

func TestUserService_Authenticate_RejectsEmptyPassword(t *testing.T) {
	svc, _, _ := setupUserSvc()

	result, err := svc.Authenticate(context.Background(), "admin@zeus.com", "")
	assert.ErrorIs(t, err, service.ErrEmptyPassword)
	assert.Nil(t, result)
}
