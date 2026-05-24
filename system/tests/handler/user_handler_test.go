package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zeus-system-service/internal/handler"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUserTest() (*gin.Engine, *handler.MockUserService) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(handler.MockUserService)
	h := handler.NewUserHandler(mockSvc)
	r := gin.New()

	users := r.Group("/users")
	{
		users.POST("", h.Create)
		users.GET("", h.List)
		users.GET("/:id", h.GetByID)
		users.PUT("/:id", h.Update)
		users.PATCH("/:id/status", h.SetStatus)
	}

	return r, mockSvc
}

func TestUserHandler_Create_201(t *testing.T) {
	r, mockSvc := setupUserTest()

	req := models.CreateUserRequest{
		Email:    "new@zeus.com",
		Password: "securepass123",
		FullName: "New User",
		Role:     "scm_operator",
	}
	created := &models.User{
		ID:       uuid.New(),
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		Status:   models.AccountStatusActive,
	}
	body, _ := json.Marshal(req)

	mockSvc.On("Create", mock.Anything, mock.AnythingOfType("models.CreateUserRequest")).Return(created, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/users", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusCreated, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 201, env.StatusCode)
	var data models.UserResponse
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, data.ID)
	assert.Equal(t, created.Email, data.Email)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Create_400_InvalidBody(t *testing.T) {
	r, _ := setupUserTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/users", bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Create_400_ValidationError(t *testing.T) {
	r, mockSvc := setupUserTest()

	req := models.CreateUserRequest{Email: "", Password: "", FullName: "", Role: ""}
	body, _ := json.Marshal(req)

	mockSvc.On("Create", mock.Anything, mock.AnythingOfType("models.CreateUserRequest")).
		Return(nil, service.ErrEmptyEmail)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/users", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Create_409_Duplicate(t *testing.T) {
	r, mockSvc := setupUserTest()

	req := models.CreateUserRequest{
		Email:    "dup@zeus.com",
		Password: "securepass123",
		FullName: "Dup",
		Role:     "sales_worker",
	}
	body, _ := json.Marshal(req)

	mockSvc.On("Create", mock.Anything, mock.AnythingOfType("models.CreateUserRequest")).
		Return(nil, service.ErrDuplicateEmail)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/users", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_GetByID_200(t *testing.T) {
	r, mockSvc := setupUserTest()
	id := uuid.New()
	user := &models.User{ID: id, Email: "user@zeus.com", FullName: "User", Role: "sales_worker", Status: models.AccountStatusActive}

	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/users/"+id.String(), nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data models.UserResponse
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Equal(t, id, data.ID)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_GetByID_404(t *testing.T) {
	r, mockSvc := setupUserTest()
	id := uuid.New()

	mockSvc.On("GetByID", mock.Anything, id).Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/users/"+id.String(), nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_GetByID_400_InvalidUUID(t *testing.T) {
	r, _ := setupUserTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/users/not-a-uuid", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_List_200(t *testing.T) {
	r, mockSvc := setupUserTest()
	users := []models.User{
		{ID: uuid.New(), Email: "a@z.com", FullName: "A", Role: "sales_worker", Status: models.AccountStatusActive},
		{ID: uuid.New(), Email: "b@z.com", FullName: "B", Role: "scm_operator", Status: models.AccountStatusActive},
	}
	meta := &models.PaginationMeta{Page: 1, Limit: 15, TotalRows: 2, TotalPages: 1}

	mockSvc.On("List", mock.Anything, 1, 15, "").Return(users, meta, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/users", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var env struct {
		StatusCode int             `json:"statusCode"`
		Data       json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 200, env.StatusCode)
	var data struct {
		Items      []models.UserResponse  `json:"items"`
		Pagination models.PaginationMeta  `json:"pagination"`
	}
	err = json.Unmarshal(env.Data, &data)
	assert.NoError(t, err)
	assert.Len(t, data.Items, 2)
	assert.Equal(t, 1, data.Pagination.TotalPages)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Update_200(t *testing.T) {
	r, mockSvc := setupUserTest()
	id := uuid.New()
	name := "Updated Name"
	role := "admin"
	req := models.UpdateUserRequest{FullName: &name, Role: &role}
	body, _ := json.Marshal(req)
	updated := &models.User{ID: id, FullName: name, Role: role}

	mockSvc.On("Update", mock.Anything, id, mock.AnythingOfType("models.UpdateUserRequest")).Return(updated, nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PUT", "/users/"+id.String(), bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Update_400(t *testing.T) {
	r, _ := setupUserTest()
	id := uuid.New()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PUT", "/users/"+id.String(), bytes.NewReader([]byte(`not json`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Update_404(t *testing.T) {
	r, mockSvc := setupUserTest()
	id := uuid.New()
	body, _ := json.Marshal(models.UpdateUserRequest{})

	mockSvc.On("Update", mock.Anything, id, mock.AnythingOfType("models.UpdateUserRequest")).Return(nil, service.ErrNotFound)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PUT", "/users/"+id.String(), bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_SetStatus_200(t *testing.T) {
	r, mockSvc := setupUserTest()
	id := uuid.New()

	mockSvc.On("SetStatus", mock.Anything, id, models.AccountStatusInactive).Return(nil)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PATCH", "/users/"+id.String()+"/status", bytes.NewReader([]byte(`{"status":"INACTIVE"}`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_SetStatus_400_InvalidStatus(t *testing.T) {
	r, _ := setupUserTest()
	id := uuid.New()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PATCH", "/users/"+id.String()+"/status", bytes.NewReader([]byte(`{"status":"INVALID"}`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_SetStatus_404(t *testing.T) {
	r, mockSvc := setupUserTest()
	id := uuid.New()

	mockSvc.On("SetStatus", mock.Anything, id, models.AccountStatusInactive).Return(service.ErrNotFound)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PATCH", "/users/"+id.String()+"/status", bytes.NewReader([]byte(`{"status":"INACTIVE"}`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_SetStatus_400_InvalidUUID(t *testing.T) {
	r, _ := setupUserTest()

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PATCH", "/users/not-a-uuid/status", bytes.NewReader([]byte(`{"status":"INACTIVE"}`)))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
