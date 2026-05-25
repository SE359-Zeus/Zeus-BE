package valkey

import (
	"context"
	"fmt"
	"strings"

	rootrepo "zeus-sales-service/internal/repository"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	addr          string
	queueKey      string
	payloadKey    string
	atpPrefix     string
	reservePrefix string
}

func New(addr string) *Repository {
	return &Repository{
		addr:          strings.TrimSpace(addr),
		queueKey:      "sales:allocation_queue",
		payloadKey:    "sales:allocation_queue:payload",
		atpPrefix:     "sales:atp:",
		reservePrefix: "sales:reservation:",
	}
}

func (repo *Repository) withClient(ctx context.Context, fn func(*redis.Client) error) error {
	if repo == nil || repo.addr == "" {
		return fmt.Errorf("valkey address is empty")
	}
	client := redis.NewClient(&redis.Options{Addr: repo.addr})
	defer client.Close()
	return fn(client)
}

var _ rootrepo.ValkeyRepository = (*Repository)(nil)
