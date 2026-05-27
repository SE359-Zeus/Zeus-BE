package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------
// Original table-driven test (kept intact)
// ------------------------------------------------------------

func TestProductionService_PlanProduction_TableDriven(t *testing.T) {
	svc := NewProductionService(setupMockRepo())

	tests := []struct {
		name          string
		request       models.CreateProductionOrderRequest
		expectError   bool
		expectedModel string
	}{
		{
			name: "Valid Request",
			request: models.CreateProductionOrderRequest{
				ProductModelCode: "MODEL-A",
				TargetQuantity:   100,
			},
			expectError:   false,
			expectedModel: "MODEL-A",
		},
		{
			name: "Zero Quantity - Should Fail",
			request: models.CreateProductionOrderRequest{
				ProductModelCode: "MODEL-B",
				TargetQuantity:   0,
			},
			expectError: true,
		},
		{
			name: "Negative Quantity - Should Fail",
			request: models.CreateProductionOrderRequest{
				ProductModelCode: "MODEL-C",
				TargetQuantity:   -50,
			},
			expectError: true,
		},
		{
			name: "Extreme Quantity - Should Fail or Cap",
			request: models.CreateProductionOrderRequest{
				ProductModelCode: "MODEL-D",
				TargetQuantity:   999999999,
			},
			expectError: true,
		},
		{
			name: "Missing Model Code - Should Fail",
			request: models.CreateProductionOrderRequest{
				ProductModelCode: "",
				TargetQuantity:   10,
			},
			expectError: true,
		},
		{
			name: "Invalid Characters in Model Code - Should Fail",
			request: models.CreateProductionOrderRequest{
				ProductModelCode: "MODEL*&^%",
				TargetQuantity:   10,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.PlanProduction(context.Background(), tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, tt.expectedModel, res.ProductModelCode)
			}
		})
	}
}

// ------------------------------------------------------------
// Hard: response must echo every field from the request
// ------------------------------------------------------------

func TestPlanProduction_EchoesAllRequestFields(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateProductionOrderRequest{
		ProductModelCode: "MODEL-ECHO",
		TargetQuantity:   42,
		ScheduledAt:      time.Now().UTC().Truncate(time.Second),
	}
	res, err := svc.PlanProduction(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "MODEL-ECHO", res.ProductModelCode, "ProductModelCode must be echoed from request")
	assert.Equal(t, 42, res.TargetQuantity, "TargetQuantity must be echoed from request")
}

// Hard: stub returns uuid.Nil — a real impl must mint a fresh ID
func TestPlanProduction_MintsFreshUUID(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateProductionOrderRequest{ProductModelCode: "MODEL-UUID", TargetQuantity: 1}
	res, err := svc.PlanProduction(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotEqual(t, uuid.Nil, res.ID, "Response must carry a non-nil UUID")
}

// Hard: stub returns the same zero value every time — IDs must diverge
func TestPlanProduction_IDsAreUnique(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateProductionOrderRequest{ProductModelCode: "MODEL-UNIQ", TargetQuantity: 5}

	res1, err := svc.PlanProduction(context.Background(), req)
	require.NoError(t, err)
	res2, err := svc.PlanProduction(context.Background(), req)
	require.NoError(t, err)

	assert.NotEqual(t, res1.ID, res2.ID, "Each call must produce a distinct order ID")
}

// Hard: stub ignores all validation — every case below must return an error
func TestPlanProduction_ValidationRules(t *testing.T) {
	svc := NewProductionService(setupMockRepo())

	cases := []struct {
		label string
		req   models.CreateProductionOrderRequest
	}{
		{"zero quantity", models.CreateProductionOrderRequest{ProductModelCode: "M", TargetQuantity: 0}},
		{"negative quantity", models.CreateProductionOrderRequest{ProductModelCode: "M", TargetQuantity: -1}},
		{"empty model code", models.CreateProductionOrderRequest{ProductModelCode: "", TargetQuantity: 10}},
		{"whitespace model code", models.CreateProductionOrderRequest{ProductModelCode: "   ", TargetQuantity: 10}},
		{"special chars in code", models.CreateProductionOrderRequest{ProductModelCode: "M*&^%!", TargetQuantity: 10}},
		{"quantity overflow", models.CreateProductionOrderRequest{ProductModelCode: "M", TargetQuantity: 999_999_999}},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			res, err := svc.PlanProduction(context.Background(), c.req)
			assert.Error(t, err, "expected validation error")
			assert.Nil(t, res, "response must be nil on validation failure")
		})
	}
}

// Hard: stub returns empty string "" which is not a valid status constant
func TestPlanProduction_InitialStatusIsSet(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	req := models.CreateProductionOrderRequest{ProductModelCode: "MODEL-STATUS", TargetQuantity: 10}
	res, err := svc.PlanProduction(context.Background(), req)
	require.NoError(t, err)

	validStatuses := map[models.ProductionOrderStatus]bool{
		models.StatusClearToBuild: true,
		models.StatusPartial:      true,
		models.StatusShortage:     true,
	}
	assert.True(t, validStatuses[res.Status],
		"initial status must be one of the defined constants, got %q", res.Status)
}

// Hard: stub ignores context — must propagate context.Canceled
func TestPlanProduction_CancelledContext(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := models.CreateProductionOrderRequest{ProductModelCode: "MODEL-CTX", TargetQuantity: 1}
	_, err := svc.PlanProduction(ctx, req)
	assert.Error(t, err, "cancelled context must cause an error")
	assert.ErrorIs(t, err, context.Canceled)
}

// Hard: 50 concurrent calls must all succeed and produce unique IDs with no data races
// Run with: go test -race
func TestPlanProduction_ConcurrentCallsAreRaceFree(t *testing.T) {
	svc := NewProductionService(setupMockRepo())
	const goroutines = 50

	var wg sync.WaitGroup
	errors := make(chan error, goroutines)
	ids := make(chan uuid.UUID, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := models.CreateProductionOrderRequest{
				ProductModelCode: "MODEL-RACE",
				TargetQuantity:   1,
			}
			res, err := svc.PlanProduction(context.Background(), req)
			if err != nil {
				errors <- err
				return
			}
			ids <- res.ID
		}()
	}

	wg.Wait()
	close(errors)
	close(ids)

	for err := range errors {
		t.Errorf("unexpected error in concurrent call: %v", err)
	}

	seen := make(map[uuid.UUID]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate order ID produced under concurrency: %s", id)
		}
		seen[id] = true
	}
}

func TestPlanProduction_ShortageCreationAndSCMAlerting(t *testing.T) {
	db := new(MockMRPRepository)
	scmClient := new(MockSCMClient)
	auditPub := new(MockAuditPublisher)
	svc := NewProductionService(db, scmClient, auditPub)

	partID := uuid.New()
	modelCode := "MODEL-X"
	bomEntries := []models.BomEntry{
		{
			ID:                      1,
			ParentModelCode:         modelCode,
			ComponentPartID:         partID,
			RequiredQuantityPerUnit: 3,
		},
	}

	// 10 units of modelCode requires 30 parts. Available inventory: 10 parts. Deficit = 20 parts.
	db.On("CreateProductionOrder", mock.Anything, mock.MatchedBy(func(order *models.ProductionOrder) bool {
		return order.ProductModelCode == modelCode && order.TargetQuantity == 10
	})).Return(nil)

	db.On("GetProductionOrder", mock.Anything, mock.Anything).Return(&models.ProductionOrder{
		ID:               uuid.Nil, // will be matched or overwritten
		ProductModelCode: modelCode,
		TargetQuantity:   10,
		Status:           models.StatusPlanned,
	}, nil)

	db.On("GetBOMByModelCode", mock.Anything, modelCode).Return(bomEntries, nil)

	// SCM details mapping
	scmClient.On("GetPartCatalogByID", mock.Anything, partID).Return(&models.Part{
		ID:       partID,
		SKU:      "COMP-SKU-100",
		Price:    1.5,
		StockQty: 10,
	}, nil)

	// Shortage logging expectation
	db.On("CreateShortageLog", mock.Anything, mock.MatchedBy(func(log *models.ShortageLog) bool {
		return log.PartID == partID && log.ShortageQty == 20 && log.ResolutionStatus == models.ResolutionStatusShortage
	})).Return(nil)

	// SCM RabbitMQ alert publication expectation
	auditPub.On("PublishJSON", messaging.DeficitPoolQueue, mock.MatchedBy(func(payload any) bool {
		m, ok := payload.(map[string]any)
		if !ok {
			return false
		}
		return m["sku"] == "COMP-SKU-100" && m["qty"] == 20
	})).Return(nil)

	// Status update to SHORTAGE/PARTIAL
	db.On("UpdateProductionOrderStatus", mock.Anything, mock.Anything, models.StatusPartial).Return(nil)

	req := models.CreateProductionOrderRequest{
		ProductModelCode: modelCode,
		TargetQuantity:   10,
	}

	res, err := svc.PlanProduction(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, models.StatusPartial, res.Status)
	assert.Len(t, res.Shortages, 1)
	assert.Equal(t, 20, res.Shortages[0].ShortageQty)

	db.AssertExpectations(t)
	scmClient.AssertExpectations(t)
	auditPub.AssertExpectations(t)
}
