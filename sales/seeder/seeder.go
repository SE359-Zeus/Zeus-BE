package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository"
	"zeus-sales-service/internal/service"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func SeedAll(ctx context.Context, sqliteRepo repository.DbRepository, manifestPath string) error {
	if sqliteRepo == nil {
		return fmt.Errorf("sqlite repository is required")
	}
	manifest, err := loadSCMManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load scm manifest: %w", err)
	}

	log.Println("Starting Sales Seeder...")
	services := service.NewServices(sqliteRepo, nil)

	clients, err := sqliteRepo.ListClients(ctx)
	if err != nil {
		return err
	}
	orders, err := sqliteRepo.ListOrders(ctx)
	if err != nil {
		return err
	}
	if len(clients) > 0 || len(orders) > 0 {
		log.Println("Sales data already exists, skipping seed.")
		return nil
	}

	seedClients := buildManifestClients(manifest)
	if err := seedClientsToRepository(ctx, sqliteRepo, seedClients); err != nil {
		return err
	}

	now := time.Now().UTC()
	seedOrders := buildManifestOrders(manifest, seedClients, now)

	for _, seedData := range seedOrders {
		orderCtx := context.WithValue(ctx, middlewares.ContextKeyRole, "client")
		orderCtx = context.WithValue(orderCtx, middlewares.ContextKeyUserID, seedData.ClientID)
		if _, err := services.Orders.CreateOrder(orderCtx, seedData.Request); err != nil {
			return fmt.Errorf("seed order for client %s: %w", seedData.ClientID, err)
		}
	}

	log.Println("Sales Seeder finished successfully.")
	return nil
}

type scmManifest struct {
	SchemaVersion int               `json:"schema_version"`
	Source        string            `json:"source"`
	PartCatalogs  []scmPartCatalog  `json:"part_catalogs"`
	Products      []scmProduct      `json:"products"`
	ProductModels []scmProductModel `json:"product_models"`
}

type scmPartCatalog struct {
	ID         string `json:"id"`
	PartNumber string `json:"part_number"`
	MfgNumber  string `json:"mfg_number"`
}

type scmProduct struct {
	ID               string `json:"id"`
	ProductModelCode string `json:"product_model_code"`
	CustomerID       string `json:"customer_id"`
}

type scmProductModel struct {
	ModelCode string              `json:"model_code"`
	Bom       []scmProductBomLine `json:"bom"`
}

type scmProductBomLine struct {
	PartCatalogID string `json:"part_catalog_id"`
	PartNumber    string `json:"part_number"`
	Quantity      int32  `json:"quantity"`
}

func loadSCMManifest(path string) (*scmManifest, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("manifest path is required")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest scmManifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

type manifestClientSeed struct {
	ID        uuid.UUID
	Name      string
	Tier      models.ClientTier
	Address   string
	ModelCode string
}

func buildManifestClients(manifest *scmManifest) []manifestClientSeed {
	if manifest == nil {
		return nil
	}
	seen := map[string]struct{}{}
	clients := make([]manifestClientSeed, 0, len(manifest.Products))
	addressBook := []string{
		"123 Innovation Drive, Silicon Valley, CA 94043",
		"55 Market St, San Jose, CA 95113",
		"88 Elm Avenue, Sacramento, CA 95814",
		"4200 Industry Way, Fremont, CA 94538",
		"701 Mission Blvd, Los Angeles, CA 90017",
		"9020 Harbor Point, San Diego, CA 92101",
		"9020 Harbor Point, San Diego, CA 92101",
	}
	for i, product := range manifest.Products {
		if _, ok := seen[product.CustomerID]; ok {
			continue
		}
		seen[product.CustomerID] = struct{}{}
		clientID, err := uuid.Parse(product.CustomerID)
		if err != nil {
			continue
		}
		clientName := fmt.Sprintf("SCM Customer %02d", len(clients)+1)
		if product.ProductModelCode != "" {
			clientName = fmt.Sprintf("SCM Customer %02d %s", len(clients)+1, product.ProductModelCode)
		}
		tier := models.ClientTierB2C
		if len(clients)%2 == 0 {
			tier = models.ClientTierB2B
		}
		clients = append(clients, manifestClientSeed{
			ID:        clientID,
			Name:      clientName,
			Tier:      tier,
			Address:   addressBook[i%len(addressBook)],
			ModelCode: product.ProductModelCode,
		})
	}
	if len(clients) == 0 {
		clients = append(clients, manifestClientSeed{
			ID:        uuid.New(),
			Name:      "SCM Customer 01",
			Tier:      models.ClientTierB2B,
			Address:   addressBook[0],
			ModelCode: "",
		})
	}
	return clients
}

func seedClientsToRepository(ctx context.Context, sqliteRepo repository.DbRepository, clients []manifestClientSeed) error {
	for i, client := range clients {
		if _, err := sqliteRepo.GetClientByName(ctx, client.Name); err == nil {
			continue
		}
		// Show/Define the raw API keys in the code clearly before hashing and storing.
		// Format: clnt<index>_p.secretkey<index>_secure_token
		// Example for SCM Customer 01:
		// Raw API Key: clnt01_p.secretkey01_secure_token
		// Prefix:      clnt01_p (first 8 characters)
		// Secret:      clnt01_p.secretkey01_secure_token (full string used in bcrypt)
		clientIndex := i + 1
		prefix := fmt.Sprintf("clnt%02d_p", clientIndex)
		rawApiKey := fmt.Sprintf("%s.secretkey%02d_secure_token", prefix, clientIndex)

		log.Printf("Seeding client: %s | Prefix: %s | Raw API Key: %s", client.Name, prefix, rawApiKey)

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(rawApiKey), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash api key: %w", err)
		}

		if err := sqliteRepo.CreateClient(ctx, &models.Client{
			ID:                        client.ID,
			Name:                      client.Name,
			Tier:                      client.Tier,
			DefaultDestinationAddress: client.Address,
			ApiKeyPrefix:              prefix,
			ApiKeyHash:                string(hashBytes),
		}); err != nil {
			return fmt.Errorf("seed client %s: %w", client.Name, err)
		}
	}
	return nil
}

type seedOrderPayload struct {
	ClientID uuid.UUID
	Request  models.CreateOrderRequest
}

func buildManifestOrders(manifest *scmManifest, clients []manifestClientSeed, now time.Time) []seedOrderPayload {
	if manifest == nil || len(clients) == 0 {
		return nil
	}
	skuPool := uniqueManifestSKUs(manifest.PartCatalogs)
	if len(skuPool) == 0 {
		return nil
	}
	orders := make([]seedOrderPayload, 0, len(clients)*2)
	for i, client := range clients {
		orderCount := 1
		if i%2 == 0 {
			orderCount = 2
		}
		for j := 0; j < orderCount; j++ {
			items := buildOrderItemsFromManifest(skuPool, i, j)
			orders = append(orders, seedOrderPayload{
				ClientID: client.ID,
				Request: models.CreateOrderRequest{
					RequiredDate: now.Add(time.Duration(36+(i*12)+(j*8)) * time.Hour),
					Items:        items,
				},
			})
		}
	}
	return orders
}

func uniqueManifestSKUs(partCatalogs []scmPartCatalog) []string {
	seen := make(map[string]struct{}, len(partCatalogs))
	skus := make([]string, 0, len(partCatalogs))
	for _, catalog := range partCatalogs {
		if catalog.PartNumber == "" {
			continue
		}
		if _, ok := seen[catalog.PartNumber]; ok {
			continue
		}
		seen[catalog.PartNumber] = struct{}{}
		skus = append(skus, catalog.PartNumber)
	}
	sort.Strings(skus)
	return skus
}

func buildOrderItemsFromManifest(skus []string, orderIndex int, batchIndex int) []models.OrderItemRequest {
	if len(skus) == 0 {
		return nil
	}
	itemCount := 2
	if (orderIndex+batchIndex)%3 == 0 {
		itemCount = 3
	}
	items := make([]models.OrderItemRequest, 0, itemCount)
	for i := 0; i < itemCount; i++ {
		sku := skus[(orderIndex+batchIndex+i)%len(skus)]
		items = append(items, models.OrderItemRequest{
			SKU:          sku,
			RequestedQty: 1 + ((orderIndex + batchIndex + i) % 6),
		})
	}
	return items
}
