package scm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient() *Client {
	baseURL := os.Getenv("SCM_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	apiKey := os.Getenv("X_API_KEY")
	if apiKey == "" {
		apiKey = "scmkey01-admin-20260524"
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.Part, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/sku/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SCM API returned status %d", resp.StatusCode)
	}

	var res struct {
		ID          uuid.UUID `json:"id"`
		SKU         string    `json:"sku"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &models.Part{
		ID:          res.ID,
		SKU:         res.SKU,
		Description: res.Description,
		Price:       res.Price,
	}, nil
}

func (c *Client) CreateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error) {
	body := map[string]any{
		"sku":         sku,
		"description": description,
		"price":       price,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create catalog part in SCM: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var res struct {
		ID          uuid.UUID `json:"id"`
		SKU         string    `json:"sku"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &models.Part{
		ID:          res.ID,
		SKU:         res.SKU,
		Description: res.Description,
		Price:       res.Price,
	}, nil
}

func (c *Client) UpdateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error) {
	body := map[string]any{
		"description": description,
		"price":       price,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, "PUT", urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("component with SKU %s not found", sku)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update catalog part in SCM: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var res struct {
		ID          uuid.UUID `json:"id"`
		SKU         string    `json:"sku"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &models.Part{
		ID:          res.ID,
		SKU:         res.SKU,
		Description: res.Description,
		Price:       res.Price,
	}, nil
}

func (c *Client) DeleteCatalogPart(ctx context.Context, sku string) error {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, "DELETE", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("component with SKU %s not found", sku)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete catalog part in SCM: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}
