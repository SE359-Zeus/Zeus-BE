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
	"zeus-mrp-service/internal/infrastructure/observability"
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

func propagateTrace(ctx context.Context, req *http.Request) {
	traceID := observability.TraceIDFromContext(ctx)
	if traceID == "" {
		return
	}
	spanID := observability.NewSpanID()
	req.Header.Set("traceparent", "00-"+traceID+"-"+spanID+"-01")
}

func (c *Client) GetPartCatalogBySKU(ctx context.Context, sku string) (*models.Part, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/sku/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

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

	var envelope struct {
		Data struct {
			ID          uuid.UUID `json:"id"`
			SKU         string    `json:"sku"`
			Description string    `json:"description"`
			Price       float64   `json:"price"`
			StockQty    int       `json:"stock_qty"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Data.ID == uuid.Nil {
		return nil, nil
	}
	return &models.Part{ID: envelope.Data.ID, SKU: envelope.Data.SKU, Description: envelope.Data.Description, Price: envelope.Data.Price, StockQty: envelope.Data.StockQty}, nil
}

func (c *Client) GetPartCatalogByID(ctx context.Context, id uuid.UUID) (*models.Part, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/%s", c.baseURL, url.PathEscape(id.String()))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

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

	var envelope struct {
		Data struct {
			ID          uuid.UUID `json:"id"`
			PartNumber  string    `json:"part_number"`
			Description string    `json:"description"`
			Price       float64   `json:"price"`
			StockQty    int       `json:"stock_qty"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Data.ID == uuid.Nil {
		return nil, nil
	}
	return &models.Part{ID: envelope.Data.ID, SKU: envelope.Data.PartNumber, Description: envelope.Data.Description, Price: envelope.Data.Price, StockQty: envelope.Data.StockQty}, nil
}


func (c *Client) GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/stocks/%s", c.baseURL, url.PathEscape(sku))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

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

	var envelope struct {
		Data models.ComponentStock `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (c *Client) GetProductModelByCode(ctx context.Context, code string) (*models.ProductModel, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/product-models/%s", c.baseURL, url.PathEscape(code))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

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

	var envelope struct {
		Data models.ProductModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (c *Client) ListStocks(ctx context.Context, page, limit int, sortBy, sortDir, q string) ([]models.ComponentStock, bool, error) {
	query := url.Values{}
	if page > 0 {
		query.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if sortBy != "" {
		query.Set("sort_by", sortBy)
	}
	if sortDir != "" {
		query.Set("sort_dir", sortDir)
	}
	if q != "" {
		query.Set("q", q)
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/stocks?%s", c.baseURL, query.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("SCM API returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Message    string `json:"message"`
		StatusCode int    `json:"statusCode"`
		Metadata   struct {
			Pagination struct {
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				TotalRows  int `json:"total_rows"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		} `json:"metadata"`
		Data []models.ComponentStock `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, false, err
	}

	hasMore := envelope.Metadata.Pagination.TotalPages > 0 && envelope.Metadata.Pagination.Page < envelope.Metadata.Pagination.TotalPages
	return envelope.Data, hasMore, nil
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
	propagateTrace(ctx, req)

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
	propagateTrace(ctx, req)

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
	propagateTrace(ctx, req)

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

func (c *Client) GetInventoryLedger(ctx context.Context, page, limit int, sortBy, sortDir, txnType, sku string) ([]models.InventoryLedgerEntry, bool, error) {
	query := url.Values{}
	if page > 0 {
		query.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if sortBy != "" {
		query.Set("sort_by", sortBy)
	}
	if sortDir != "" {
		query.Set("sort_dir", sortDir)
	}
	if txnType != "" {
		query.Set("type", txnType)
	}
	if sku != "" {
		query.Set("sku", sku)
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/ledger?%s", c.baseURL, query.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("SCM API returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Message    string `json:"message"`
		StatusCode int    `json:"statusCode"`
		Metadata   struct {
			Pagination struct {
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				TotalRows  int `json:"total_rows"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		} `json:"metadata"`
		Data []models.InventoryLedgerEntry `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, false, err
	}

	hasMore := envelope.Metadata.Pagination.TotalPages > 0 && envelope.Metadata.Pagination.Page < envelope.Metadata.Pagination.TotalPages
	return envelope.Data, hasMore, nil
}

func (c *Client) GetLUTs(ctx context.Context) (*models.LUTCollection, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/luts", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SCM API returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Data models.LUTCollection `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (c *Client) GetOptimalSupplier(ctx context.Context, sku string) (uuid.UUID, float64, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/vendors/optimal?sku=%s", c.baseURL, url.QueryEscape(sku))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return uuid.Nil, 0, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return uuid.Nil, 0, fmt.Errorf("no optimal supplier found for sku: %s", sku)
	}
	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, 0, fmt.Errorf("SCM API returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Data struct {
			Supplier struct {
				ID uuid.UUID `json:"id"`
			} `json:"supplier"`
			Mapping struct {
				Price float64 `json:"price"`
			} `json:"mapping"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return uuid.Nil, 0, err
	}

	if envelope.Data.Supplier.ID == uuid.Nil {
		return uuid.Nil, 0, fmt.Errorf("optimal supplier ID is nil")
	}

	return envelope.Data.Supplier.ID, envelope.Data.Mapping.Price, nil
}

func (c *Client) CreateDraftPO(ctx context.Context, vendorID uuid.UUID, targetBuild string) (string, error) {
	body := map[string]any{
		"vendor_id":    vendorID.String(),
		"target_build": targetBuild,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/purchase-orders/draft", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create draft PO: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}

	if envelope.Data.ID == "" {
		return "", fmt.Errorf("received empty PO ID from SCM")
	}

	return envelope.Data.ID, nil
}

func (c *Client) AddLineItemWithLock(ctx context.Context, poID string, sku string, qty int) error {
	body := map[string]any{
		"sku": sku,
		"qty": qty,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/purchase-orders/%s/line-items", c.baseURL, url.PathEscape(poID))
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add line item with lock: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

func (c *Client) ListPOs(ctx context.Context, targetBuild string) ([]models.PurchaseOrder, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/purchase-orders?q=%s&limit=1000", c.baseURL, url.QueryEscape(targetBuild))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SCM API ListPOs returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope struct {
		Data []models.PurchaseOrder `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}

	return envelope.Data, nil
}

func (c *Client) CreateProductModel(ctx context.Context, code string, name string, price float64) error {
	body := map[string]any{
		"model_code": code,
		"model_name": name,
		"unit_price": price,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/product-models", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create product model in SCM: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

func (c *Client) DeleteProductModel(ctx context.Context, code string) error {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/product-models/%s", c.baseURL, url.PathEscape(code))
	req, err := http.NewRequestWithContext(ctx, "DELETE", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete product model in SCM: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return nil
}

type SCMInventoryMetrics struct {
	TotalSKUs  int     `json:"total_skus"`
	LowStock   int     `json:"low_stock"`
	OutOfStock int     `json:"out_of_stock"`
	StockValue float64 `json:"stock_value"`
}

func (c *Client) GetInventoryMetrics(ctx context.Context) (*SCMInventoryMetrics, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/metrics", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SCM inventory metrics returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Data SCMInventoryMetrics `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (c *Client) GetInventoryTransactionByID(ctx context.Context, txnID string) (*models.InventoryLedgerEntry, error) {
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/ledger/%s", c.baseURL, url.PathEscape(txnID))
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	propagateTrace(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SCM ledger lookup returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Data models.InventoryLedgerEntry `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

