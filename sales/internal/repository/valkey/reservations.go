package valkey

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (repo *Repository) ReserveInventory(ctx context.Context, orderID uuid.UUID, items []models.ReservationItem) error {
	reservationKey := repo.reservePrefix + orderID.String()
	return repo.withClient(ctx, func(clientConn *redis.Client) error {
		pipe := clientConn.TxPipeline()
		for _, item := range items {
			key := repo.atpPrefix + strings.ToUpper(strings.TrimSpace(item.SKU))
			current, err := repo.GetATP(ctx, item.SKU)
			if err != nil {
				return err
			}
			if current < item.Quantity {
				return fmt.Errorf("insufficient inventory for %s", item.SKU)
			}
			pipe.DecrBy(ctx, key, int64(item.Quantity))
			pipe.HSet(ctx, reservationKey, strings.ToUpper(strings.TrimSpace(item.SKU)), item.Quantity)
		}
		_, err := pipe.Exec(ctx)
		return err
	})
}

func (repo *Repository) ReleaseInventory(ctx context.Context, orderID uuid.UUID) error {
	reservationKey := repo.reservePrefix + orderID.String()
	return repo.withClient(ctx, func(clientConn *redis.Client) error {
		entries, err := clientConn.HGetAll(ctx, reservationKey).Result()
		if err != nil {
			return err
		}
		pipe := clientConn.TxPipeline()
		for sku, quantityText := range entries {
			quantity, err := strconv.Atoi(quantityText)
			if err != nil {
				return err
			}
			pipe.IncrBy(ctx, repo.atpPrefix+sku, int64(quantity))
		}
		pipe.Del(ctx, reservationKey)
		_, err = pipe.Exec(ctx)
		return err
	})
}
