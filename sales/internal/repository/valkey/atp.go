package valkey

import (
	"context"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

func (repo *Repository) GetATP(ctx context.Context, sku string) (int, error) {
	value, err := repo.client.Get(ctx, repo.atpPrefix+strings.ToUpper(strings.TrimSpace(sku))).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func (repo *Repository) SetATP(ctx context.Context, sku string, quantity int) error {
	return repo.client.Set(ctx, repo.atpPrefix+strings.ToUpper(strings.TrimSpace(sku)), quantity, 0).Err()
}
