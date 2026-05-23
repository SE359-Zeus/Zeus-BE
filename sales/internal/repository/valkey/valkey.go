package valkey

import (
	rootrepo "zeus-sales-service/internal/repository"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	client        *redis.Client
	queueKey      string
	payloadKey    string
	atpPrefix     string
	reservePrefix string
}

func (repo *Repository) Close() error {
	if repo == nil || repo.client == nil {
		return nil
	}
	return repo.client.Close()
}

func New(client *redis.Client) *Repository {
	return &Repository{
		client:        client,
		queueKey:      "sales:allocation_queue",
		payloadKey:    "sales:allocation_queue:payload",
		atpPrefix:     "sales:atp:",
		reservePrefix: "sales:reservation:",
	}
}

var _ rootrepo.ValkeyRepository = (*Repository)(nil)
