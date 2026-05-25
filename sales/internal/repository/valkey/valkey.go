package valkey

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"time"

	rootrepo "zeus-sales-service/internal/repository"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	addr          string
	disabled      atomic.Bool
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
	if repo == nil || repo.addr == "" || repo.disabled.Load() {
		return nil
	}
	if err := probeTCP(repo.addr); err != nil {
		repo.disabled.Store(true)
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         repo.addr,
		MaxRetries:   -1,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		PoolTimeout:  500 * time.Millisecond,
	})
	defer client.Close()
	return fn(client)
}

func probeTCP(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
	if err != nil {
		return err
	}
	return conn.Close()
}

var _ rootrepo.ValkeyRepository = (*Repository)(nil)
