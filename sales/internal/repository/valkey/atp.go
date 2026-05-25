package valkey

import (
	"context"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

func (repo *Repository) GetATP(ctx context.Context, sku string) (int, error) {
	var quantity int
	err := repo.withClient(ctx, func(clientConn *redis.Client) error {
		value, err := clientConn.Get(ctx, repo.atpPrefix+strings.ToUpper(strings.TrimSpace(sku))).Result()
		if err != nil {
			if err == redis.Nil {
				quantity = 0
				return nil
			}
			return err
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		quantity = parsed
		return nil
	})
	return quantity, err
}

func (repo *Repository) SetATP(ctx context.Context, sku string, quantity int) error {
	return repo.withClient(ctx, func(clientConn *redis.Client) error {
		return clientConn.Set(ctx, repo.atpPrefix+strings.ToUpper(strings.TrimSpace(sku)), quantity, 0).Err()
	})
}
