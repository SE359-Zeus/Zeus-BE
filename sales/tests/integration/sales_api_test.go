package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"zeus-sales-service/internal/controllers"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository/sqlite"
	"zeus-sales-service/internal/repository/valkey"
	"zeus-sales-service/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type responseEnvelope struct {
	Message    string          `json:"message"`
	StatusCode int             `json:"statusCode"`
	Metadata   json.RawMessage `json:"metadata"`
	Data       json.RawMessage `json:"data"`
}

func TestSalesAPI_OrderLifecycleAndLocking(t *testing.T) {
	router, sqliteRepo, valkeyRepo := newIntegrationHarness(t)
	require.NoError(t, valkeyRepo.SetATP(context.Background(), "SKU-LOCK", 5))

	createBody := models.CreateOrderRequest{
		ClientName:         "Integration Client",
		DestinationAddress: "Warehouse 12",
		RequiredDate:       time.Now().Add(48 * time.Hour).UTC(),
		Items:              []models.OrderItemRequest{{SKU: "SKU-LOCK", RequestedQty: 2, UnitPrice: 11}},
	}
	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/orders", createBody)
	require.Equal(t, http.StatusCreated, createResp.Code)

	var createEnvelope responseEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &createEnvelope))
	var created models.OrderResponse
	require.NoError(t, json.Unmarshal(createEnvelope.Data, &created))
	require.NotEqual(t, created.Order.ID.String(), "")

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/orders/"+created.Order.ID.String(), nil)
	require.Equal(t, http.StatusOK, getResp.Code)

	processResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/fulfillment/process", nil)
	require.Equal(t, http.StatusOK, processResp.Code)

	patchResp := doJSONRequest(t, router, http.MethodPatch, "/api/v1/sales/orders/"+created.Order.ID.String(), models.UpdateOrderRequest{DestinationAddress: ptrString("New Dock")})
	require.Equal(t, http.StatusConflict, patchResp.Code)

	var after models.OrderResponse
	assertStatus := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/orders/"+created.Order.ID.String(), nil)
	require.Equal(t, http.StatusOK, assertStatus.Code)
	var afterEnvelope responseEnvelope
	require.NoError(t, json.Unmarshal(assertStatus.Body.Bytes(), &afterEnvelope))
	require.NoError(t, json.Unmarshal(afterEnvelope.Data, &after))
	require.NotNil(t, after.Order.Status)
	require.Equal(t, models.SalesOrderStatusProcessingCode, after.Order.Status.Code)
	require.True(t, after.Order.Locked)
	_, _ = sqliteRepo, valkeyRepo
}

func TestSalesAPI_ClientRegistryAndQueueStatus(t *testing.T) {
	router, _, _ := newIntegrationHarness(t)

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/orders", models.CreateOrderRequest{
		ClientName:   "Registry Client",
		RequiredDate: time.Now().Add(24 * time.Hour).UTC(),
		Items:        []models.OrderItemRequest{{SKU: "SKU-R", RequestedQty: 1, UnitPrice: 3}},
	})
	require.Equal(t, http.StatusCreated, createResp.Code)

	queueResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/fulfillment/queue", nil)
	require.Equal(t, http.StatusOK, queueResp.Code)

	clientsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/clients", nil)
	require.Equal(t, http.StatusOK, clientsResp.Code)

	var clientEnvelope responseEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &clientEnvelope))
	var created models.OrderResponse
	require.NoError(t, json.Unmarshal(clientEnvelope.Data, &created))
	clientResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/clients/"+created.Client.ID.String(), nil)
	require.Equal(t, http.StatusOK, clientResp.Code)
}

func TestSalesAPI_CancelOrderEndpoint(t *testing.T) {
	router, _, valkeyRepo := newIntegrationHarness(t)
	require.NoError(t, valkeyRepo.SetATP(context.Background(), "SKU-CANCEL", 4))

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/orders", models.CreateOrderRequest{
		ClientName:         "Cancel Client",
		DestinationAddress: "Dock Cancel",
		RequiredDate:       time.Now().Add(72 * time.Hour).UTC(),
		Items:              []models.OrderItemRequest{{SKU: "SKU-CANCEL", RequestedQty: 2, UnitPrice: 7}},
	})
	require.Equal(t, http.StatusCreated, createResp.Code)

	var createEnvelope responseEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &createEnvelope))
	var created models.OrderResponse
	require.NoError(t, json.Unmarshal(createEnvelope.Data, &created))

	cancelResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/orders/"+created.Order.ID.String()+"/cancel", nil)
	require.Equal(t, http.StatusOK, cancelResp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/orders/"+created.Order.ID.String(), nil)
	require.Equal(t, http.StatusOK, getResp.Code)

	var getEnvelope responseEnvelope
	require.NoError(t, json.Unmarshal(getResp.Body.Bytes(), &getEnvelope))
	var cancelled models.OrderResponse
	require.NoError(t, json.Unmarshal(getEnvelope.Data, &cancelled))
	require.NotNil(t, cancelled.Order.Status)
	require.Equal(t, models.SalesOrderStatusCancelledCode, cancelled.Order.Status.Code)
}

func TestSalesAPI_CreateOrder_AcceptsCamelCaseRequestBody(t *testing.T) {
	router, _, valkeyRepo := newIntegrationHarness(t)
	require.NoError(t, valkeyRepo.SetATP(context.Background(), "sadfawefdf", 3))

	body := []byte(`{
		"clientName": "Hung",
		"clientTier": "B2B",
		"destinationAddress": "123 Nguyen Van Troi, TPHCM",
		"items": [
			{
				"requestedQty": 1,
				"sku": "sadfawefdf",
				"unitPrice": 1
			}
		],
		"requiredDate": "2026-1-1T10:00:00"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var envelope responseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	var created models.OrderResponse
	require.NoError(t, json.Unmarshal(envelope.Data, &created))
	require.Equal(t, "Hung", created.Order.ClientName)
	require.Equal(t, models.ClientTierB2B, created.Client.Tier)
	require.NotNil(t, created.Order.Status)
	require.Equal(t, models.SalesOrderStatusPendingCode, created.Order.Status.Code)
}

func TestSalesAPI_ListOrders_ReturnsSummaryRows(t *testing.T) {
	router, _, valkeyRepo := newIntegrationHarness(t)
	require.NoError(t, valkeyRepo.SetATP(context.Background(), "SKU-SUMMARY", 3))

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/orders", models.CreateOrderRequest{
		ClientName:         "Summary Client",
		DestinationAddress: "Summary Dock",
		ClientTier:         models.ClientTierB2B,
		RequiredDate:       time.Date(2026, time.January, 3, 10, 0, 0, 0, time.UTC),
		Items:              []models.OrderItemRequest{{SKU: "SKU-SUMMARY", RequestedQty: 1, UnitPrice: 5}},
	})
	require.Equal(t, http.StatusCreated, createResp.Code)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sales/orders", nil)
	require.Equal(t, http.StatusOK, listResp.Code)

	var listEnvelope responseEnvelope
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listEnvelope))
	var rows []struct {
		OrderID      string    `json:"orderId"`
		ClientName   string    `json:"clientName"`
		RequiredDate time.Time `json:"requiredDate"`
		TotalValue   float64   `json:"totalValue"`
		Status       string    `json:"status"`
	}
	require.NoError(t, json.Unmarshal(listEnvelope.Data, &rows))
	require.NotEmpty(t, rows)
	require.Equal(t, "Summary Client", rows[0].ClientName)
	require.Equal(t, models.SalesOrderStatusPendingCode, rows[0].Status)
	require.NotZero(t, rows[0].OrderID)
}

func newIntegrationHarness(t *testing.T) (http.Handler, *sqlite.Repository, *valkey.Repository) {
	t.Helper()
	sqliteRepo, err := sqlite.Open(filepath.Join(t.TempDir(), "sales.db"))
	require.NoError(t, err)
	server := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	valkeyRepo := valkey.New(redisClient)
	t.Cleanup(func() {
		_ = redisClient.Close()
		_ = sqliteRepo.Close()
	})
	return controllers.NewMux(service.NewServices(sqliteRepo, valkeyRepo)), sqliteRepo, valkeyRepo
}

func doJSONRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		payload = encoded
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func ptrString(value string) *string {
	return &value
}
