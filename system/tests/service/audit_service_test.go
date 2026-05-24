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
)

type mockAuditRepo struct {
	mock.Mock
}

func (m *mockAuditRepo) Insert(ctx context.Context, log *models.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *mockAuditRepo) Query(ctx context.Context, filter models.AuditFilter, page, limit int) ([]models.AuditLog, int64, error) {
	args := m.Called(ctx, filter, page, limit)
	var logs []models.AuditLog
	if v := args.Get(0); v != nil {
		logs = v.([]models.AuditLog)
	}
	return logs, args.Get(1).(int64), args.Error(2)
}

func (m *mockAuditRepo) CountByAction(ctx context.Context, actionType models.ActionType, start, end time.Time) (int64, error) {
	args := m.Called(ctx, actionType, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func setupAuditSvc() (service.AuditService, *mockAuditRepo, *service.MockActionTypeService) {
	repo := new(mockAuditRepo)
	actionTypeSvc := new(service.MockActionTypeService)
	svc := service.NewAuditService(repo, actionTypeSvc)
	return svc, repo, actionTypeSvc
}

func validIngestReq() models.IngestAuditRequest {
	return models.IngestAuditRequest{
		UserID:         uuid.New(),
		UserEmail:      "user@zeus.com",
		ActionType:     "CREATE",
		TargetResource: "users/abc-123",
		Details:        "Created new user account",
		IPAddress:      "192.168.1.1",
	}
}

func TestAuditService_Ingest_Success(t *testing.T) {
	svc, repo, actionTypeSvc := setupAuditSvc()
	req := validIngestReq()

	actionTypeSvc.On("IsValid", anyCtx, models.ActionType("CREATE")).Return(true, nil)
	repo.On("Insert", anyCtx, mock.AnythingOfType("*models.AuditLog")).Return(nil)

	err := svc.Ingest(context.Background(), req)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	actionTypeSvc.AssertExpectations(t)
}

func TestAuditService_Ingest_RejectsEmptyAction(t *testing.T) {
	svc, _, _ := setupAuditSvc()
	req := validIngestReq()
	req.ActionType = ""

	err := svc.Ingest(context.Background(), req)
	assert.Error(t, err)
}

func TestAuditService_Ingest_RejectsInvalidAction(t *testing.T) {
	svc, _, actionTypeSvc := setupAuditSvc()
	req := validIngestReq()
	req.ActionType = "INVALID"

	actionTypeSvc.On("IsValid", anyCtx, models.ActionType("INVALID")).Return(false, nil)

	err := svc.Ingest(context.Background(), req)
	assert.Error(t, err)
	actionTypeSvc.AssertExpectations(t)
}

func TestAuditService_Ingest_RejectsEmptyUserID(t *testing.T) {
	svc, _, actionTypeSvc := setupAuditSvc()
	req := validIngestReq()
	req.UserID = uuid.Nil

	actionTypeSvc.On("IsValid", anyCtx, models.ActionType("CREATE")).Return(true, nil)

	err := svc.Ingest(context.Background(), req)
	assert.Error(t, err)
	actionTypeSvc.AssertExpectations(t)
}

func TestAuditService_Ingest_RejectsEmptyUserEmail(t *testing.T) {
	svc, _, actionTypeSvc := setupAuditSvc()
	req := validIngestReq()
	req.UserEmail = ""

	actionTypeSvc.On("IsValid", anyCtx, models.ActionType("CREATE")).Return(true, nil)

	err := svc.Ingest(context.Background(), req)
	assert.Error(t, err)
	actionTypeSvc.AssertExpectations(t)
}

func TestAuditService_Ingest_RejectsEmptyTarget(t *testing.T) {
	svc, _, actionTypeSvc := setupAuditSvc()
	req := validIngestReq()
	req.TargetResource = ""

	actionTypeSvc.On("IsValid", anyCtx, models.ActionType("CREATE")).Return(true, nil)

	err := svc.Ingest(context.Background(), req)
	assert.Error(t, err)
	actionTypeSvc.AssertExpectations(t)
}

func TestAuditService_Ingest_SecurityEventFlagged(t *testing.T) {
	svc, repo, actionTypeSvc := setupAuditSvc()
	req := validIngestReq()
	req.IsSecurityEvent = true
	req.ActionType = "SECURITY"

	actionTypeSvc.On("IsValid", anyCtx, models.ActionType("SECURITY")).Return(true, nil)
	repo.On("Insert", anyCtx, mock.AnythingOfType("*models.AuditLog")).Return(nil)

	err := svc.Ingest(context.Background(), req)
	assert.NoError(t, err)

	repo.AssertCalled(t, "Insert", anyCtx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.IsSecurityEvent && log.ActionType == "SECURITY"
	}))
	actionTypeSvc.AssertExpectations(t)
}

func TestAuditService_Query_ByActionType(t *testing.T) {
	svc, repo, _ := setupAuditSvc()
	action := models.ActionType("LOGIN")
	filter := models.AuditFilter{ActionType: &action}
	expected := []models.AuditLog{
		{ActionType: "LOGIN", TargetResource: "auth/login"},
	}

	repo.On("Query", anyCtx, filter, 1, 15).Return(expected, int64(1), nil)

	logs, meta, err := svc.Query(context.Background(), filter, 1, 15)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, models.ActionType("LOGIN"), logs[0].ActionType)
	assert.Equal(t, int64(1), meta.TotalRows)
	repo.AssertExpectations(t)
}

func TestAuditService_Query_ByDateRange(t *testing.T) {
	svc, repo, _ := setupAuditSvc()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	filter := models.AuditFilter{StartDate: &start, EndDate: &end}

	repo.On("Query", anyCtx, filter, 1, 15).Return([]models.AuditLog{}, int64(0), nil)

	logs, meta, err := svc.Query(context.Background(), filter, 1, 15)
	assert.NoError(t, err)
	assert.NotNil(t, logs)
	assert.Len(t, logs, 0)
	assert.Equal(t, int64(0), meta.TotalRows)
	repo.AssertExpectations(t)
}

func TestAuditService_Query_ByUser(t *testing.T) {
	svc, repo, _ := setupAuditSvc()
	userID := uuid.New()
	filter := models.AuditFilter{UserID: &userID}

	repo.On("Query", anyCtx, filter, 1, 15).Return([]models.AuditLog{}, int64(0), nil)

	logs, meta, err := svc.Query(context.Background(), filter, 1, 15)
	assert.NoError(t, err)
	assert.NotNil(t, logs)
	assert.Equal(t, int64(0), meta.TotalRows)
	repo.AssertExpectations(t)
}

func TestAuditService_Query_ReturnsEmptySlice(t *testing.T) {
	svc, repo, _ := setupAuditSvc()

	repo.On("Query", anyCtx, models.AuditFilter{}, 1, 15).Return(nil, int64(0), nil)

	logs, meta, err := svc.Query(context.Background(), models.AuditFilter{}, 1, 15)
	assert.NoError(t, err)
	assert.NotNil(t, logs)
	assert.Len(t, logs, 0)
	assert.Equal(t, int64(0), meta.TotalRows)
	repo.AssertExpectations(t)
}

func TestAuditService_GetMetrics_Success(t *testing.T) {
	svc, repo, _ := setupAuditSvc()
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	repo.On("CountByAction", anyCtx, models.ActionType("LOGIN"), mock.Anything, mock.Anything).Return(int64(5), nil)
	repo.On("CountByAction", anyCtx, models.ActionType("SECURITY"), mock.Anything, mock.Anything).Return(int64(2), nil)
	repo.On("CountByAction", anyCtx, models.ActionType("UPDATE"), today, tomorrow).Return(int64(8), nil)
	repo.On("CountByAction", anyCtx, models.ActionType("DELETE"), today, tomorrow).Return(int64(3), nil)

	metrics, err := svc.GetMetrics(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(5), metrics.LoginsToday)
	assert.Equal(t, int64(2), metrics.SecurityEvents)
	assert.Equal(t, int64(11), metrics.ModificationVelocity)
	repo.AssertExpectations(t)
}

func TestAuditService_NoMutateMethods(t *testing.T) {
	svc, _, _ := setupAuditSvc()
	_ = svc

	var svcIface interface{} = svc
	_, hasDelete := svcIface.(interface{ Delete(interface{}) error })
	_, hasUpdate := svcIface.(interface{ Update(interface{}) error })
	assert.False(t, hasDelete, "AuditService must not expose a Delete method")
	assert.False(t, hasUpdate, "AuditService must not expose an Update method")
}
