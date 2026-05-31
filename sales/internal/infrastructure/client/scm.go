package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"zeus-sales-service/internal/infrastructure/observability"
)

type SCMClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewSCMClient(baseURL, apiKey string) *SCMClient {
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	if apiKey == "" {
		apiKey = "scmkey01-admin-20260524"
	}
	return &SCMClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// propagateTrace injects the W3C traceparent header onto an outbound request
// using the trace_id from the calling context and a fresh child span_id.
func propagateTrace(ctx context.Context, req *http.Request) {
	traceID := observability.TraceIDFromContext(ctx)
	if traceID == "" {
		return
	}
	spanID := observability.NewSpanID()
	req.Header.Set("traceparent", "00-"+traceID+"-"+spanID+"-01")
}

// CheckSKU queries SCM to verify if a SKU exists either as a part catalog item or as a product model code.
func (c *SCMClient) CheckSKU(ctx context.Context, sku string) (bool, error) {
	// 1. Try checking part-catalog by SKU
	partURL := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/sku/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, partURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
	}

	// 2. Try checking product-model by code
	prodModelURL := fmt.Sprintf("%s/api/v1/scm/inventory/product-models/%s", c.baseURL, url.PathEscape(sku))
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, prodModelURL, nil)
	if err != nil {
		return false, err
	}
	req2.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req2)

	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return false, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}

// GetProductModelPrice fetches the product model details from SCM and returns its unit price.
func (c *SCMClient) GetProductModelPrice(ctx context.Context, sku string) (float64, error) {
	prodModelURL := fmt.Sprintf("%s/api/v1/scm/inventory/product-models/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, prodModelURL, nil)
	if err != nil {
		return 0.0, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0.0, fmt.Errorf("SKU %s not found in SCM", sku)
	}
	if resp.StatusCode != http.StatusOK {
		return 0.0, fmt.Errorf("SCM returned status code %d", resp.StatusCode)
	}

	var envelope struct {
		Data struct {
			UnitPrice float64 `json:"unit_price"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return 0.0, err
	}

	return envelope.Data.UnitPrice, nil
}

// Ping checks the health endpoint of the SCM service to verify connectivity.
func (c *SCMClient) Ping(ctx context.Context) error {
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
