package seeder

import (
	"context"
	"fmt"
	"log"
	"time"

	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository"
	"zeus-sales-service/internal/service"
)

func SeedAll(ctx context.Context, sqliteRepo repository.DbRepository) error {
	if sqliteRepo == nil {
		return fmt.Errorf("sqlite repository is required")
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

	now := time.Now().UTC()

	seedOrders := []models.CreateOrderRequest{
		{
			ClientName:         "TechCorp Solutions",
			ClientTier:         models.ClientTierB2B,
			DestinationAddress: "123 Innovation Drive, Silicon Valley, CA 94043",
			RequiredDate:       now.Add(72 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-100", RequestedQty: 5, UnitPrice: 12.5},
				{SKU: "SKU-200", RequestedQty: 2, UnitPrice: 9.0},
			},
		},
		{
			ClientName:         "TechCorp Solutions",
			ClientTier:         models.ClientTierB2B,
			DestinationAddress: "123 Innovation Drive, Silicon Valley, CA 94043",
			RequiredDate:       now.Add(96 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-200", RequestedQty: 3, UnitPrice: 9.0},
				{SKU: "SKU-400", RequestedQty: 4, UnitPrice: 7.75},
			},
		},
		{
			ClientName:         "Bright Retail",
			ClientTier:         models.ClientTierB2C,
			DestinationAddress: "55 Market St, San Jose, CA 95113",
			RequiredDate:       now.Add(48 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-300", RequestedQty: 1, UnitPrice: 49.0},
			},
		},
		{
			ClientName:         "Nova Home Goods",
			ClientTier:         models.ClientTierB2C,
			DestinationAddress: "88 Elm Avenue, Sacramento, CA 95814",
			RequiredDate:       now.Add(36 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-500", RequestedQty: 2, UnitPrice: 24.5},
				{SKU: "SKU-100", RequestedQty: 1, UnitPrice: 12.5},
			},
		},
		{
			ClientName:         "Omni Manufacturing",
			ClientTier:         models.ClientTierB2B,
			DestinationAddress: "4200 Industry Way, Fremont, CA 94538",
			RequiredDate:       now.Add(120 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-600", RequestedQty: 6, UnitPrice: 5.2},
				{SKU: "SKU-200", RequestedQty: 8, UnitPrice: 9.0},
			},
		},
		{
			ClientName:         "Apex Field Services",
			ClientTier:         models.ClientTierB2B,
			DestinationAddress: "701 Mission Blvd, Los Angeles, CA 90017",
			RequiredDate:       now.Add(60 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-300", RequestedQty: 5, UnitPrice: 48.0},
				{SKU: "SKU-400", RequestedQty: 5, UnitPrice: 7.5},
			},
		},
		{
			ClientName:         "Bright Retail",
			ClientTier:         models.ClientTierB2C,
			DestinationAddress: "55 Market St, San Jose, CA 95113",
			RequiredDate:       now.Add(84 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-500", RequestedQty: 3, UnitPrice: 25.0},
			},
		},
		{
			ClientName:         "Summit Installations",
			ClientTier:         models.ClientTierB2B,
			DestinationAddress: "9020 Harbor Point, San Diego, CA 92101",
			RequiredDate:       now.Add(144 * time.Hour),
			Items: []models.OrderItemRequest{
				{SKU: "SKU-100", RequestedQty: 10, UnitPrice: 12.0},
				{SKU: "SKU-600", RequestedQty: 4, UnitPrice: 5.15},
			},
		},
	}

	for _, req := range seedOrders {
		if _, err := services.Orders.CreateOrder(ctx, req); err != nil {
			return fmt.Errorf("seed order for %s: %w", req.ClientName, err)
		}
	}

	log.Println("Sales Seeder finished successfully.")
	return nil
}
