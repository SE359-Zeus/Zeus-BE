package cache

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Conn struct {
	addr string
}

func New(addr string) *Conn {
	return &Conn{addr: strings.TrimSpace(addr)}
}

func Dial(addr string) (*Conn, error) {
	if addr == "" {
		return nil, fmt.Errorf("valkey address is empty")
	}
	return &Conn{addr: strings.TrimSpace(addr)}, nil
}

func (conn *Conn) withClient(ctx context.Context, fn func(*redis.Client) error) error {
	if conn == nil || conn.addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{Addr: conn.addr})
	defer client.Close()
	return fn(client)
}

func (conn *Conn) Ping(ctx context.Context) error {
	if conn == nil || conn.addr == "" {
		return fmt.Errorf("valkey address is empty")
	}
	return conn.withClient(ctx, func(client *redis.Client) error {
		return client.Ping(ctx).Err()
	})
}
