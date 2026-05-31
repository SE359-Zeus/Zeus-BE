package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"zeus-sales-service/internal/service"
)

type MRPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewMRPClient(baseURL, apiKey string) *MRPClient {
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	}
	if apiKey == "" {
		apiKey = "mrpkey01-admin-20260524"
	}
	return &MRPClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type mrpCreateProductionOrderRequest struct {
	ProductModelCode string    `json:"product_model_code"`
	TargetQuantity   int       `json:"target_quantity"`
	ScheduledAt      time.Time `json:"scheduled_at"`
}

func (c *MRPClient) CreateProductionOrder(ctx context.Context, req service.MRPCreateOrderReq) error {
	url := fmt.Sprintf("%s/api/v1/production/orders", c.baseURL)
	payload := mrpCreateProductionOrderRequest{
		ProductModelCode: req.ProductModelCode,
		TargetQuantity:   req.TargetQuantity,
		ScheduledAt:      req.ScheduledAt,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create production order in MRP: status %d", resp.StatusCode)
	}
	return nil
}

// Ping checks the health endpoint of the MRP service to verify connectivity.
func (c *MRPClient) Ping(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
var _ service.MRPClient = (*MRPClient)(nil)
