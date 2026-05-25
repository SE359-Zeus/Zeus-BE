package cache

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"
)

var ErrUnavailable = errors.New("valkey unavailable")

type ValkeyConn interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	SAdd(ctx context.Context, key, member string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	Exists(ctx context.Context, key string) (bool, error)
	HSet(ctx context.Context, key, field, value string) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	SIsMember(ctx context.Context, key, member string) (bool, error)
	Close()
}

type valkeyConn struct {
	client    valkey.Client
	available bool
	mu        sync.RWMutex
}

func DialValkey(addr string) ValkeyConn {
	conn := &valkeyConn{}
	if addr == "" {
		log.Println("Valkey cache disabled: no address configured")
		return conn
	}

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{addr}})
	if err != nil {
		log.Printf("Warning: Valkey client creation failed at %s: %v", addr, err)
		return conn
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		log.Printf("Warning: Valkey connection failed at %s: %v", addr, err)
		client.Close()
		return conn
	}

	log.Printf("Valkey connection successful at %s", addr)
	conn.client = client
	conn.available = true
	return conn
}

func (v *valkeyConn) isAvailable() bool {
	if v == nil {
		return false
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.available && v.client != nil
}

func (v *valkeyConn) Close() {
	if v == nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.client != nil {
		v.client.Close()
		v.client = nil
	}
	v.available = false
}
