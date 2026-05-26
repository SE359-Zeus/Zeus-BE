package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
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

// CheckSKU queries SCM to verify if a SKU exists either as a part catalog item or as a product model code.
func (c *SCMClient) CheckSKU(ctx context.Context, sku string) (bool, error) {
	// 1. Try checking part-catalog by SKU
	partURL := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/sku/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, partURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)

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
