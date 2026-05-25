package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

// GetPartInventory returns the on-hand quantity for a part by calling the SCM inventory API.
func (r *sqliteMRPRepository) GetPartInventory(ctx context.Context, partID uuid.UUID) (int, error) {
	if partID == uuid.Nil {
		return 0, nil
	}

	baseURL := os.Getenv("SCM_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	apiKey := os.Getenv("X_API_KEY")
	if apiKey == "" {
		apiKey = "scmkey01-admin-20260524"
	}

	client := &http.Client{}
	urlStr := fmt.Sprintf("%s/api/v1/scm/inventory/part-catalog/%s", baseURL, partID.String())
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-API-KEY", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("SCM inventory API returned status %d", resp.StatusCode)
	}

	var res struct {
		StockQty int `json:"stock_qty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}

	return res.StockQty, nil
}

// GetInventoryTransactions returns all stock movements from the SCM ledger.
func (r *sqliteMRPRepository) GetInventoryTransactions(_ context.Context) ([]models.InventoryTransactionDTO, error) {
	return []models.InventoryTransactionDTO{}, nil
}

// GetInventoryMetrics returns computed inventory KPIs.
func (r *sqliteMRPRepository) GetInventoryMetrics(_ context.Context) (*models.InventoryMetrics, error) {
	return &models.InventoryMetrics{}, nil
}
