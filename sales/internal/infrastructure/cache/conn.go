package cache

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type Conn struct {
	addr     string
	disabled atomic.Bool
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
	if conn == nil || conn.addr == "" || conn.disabled.Load() {
		return nil
	}
	if err := probeTCP(conn.addr); err != nil {
		conn.disabled.Store(true)
		slog.Warn("valkey connection disabled", slog.String("service", "sales"), slog.String("component", "valkey"), slog.String("addr", conn.addr), slog.String("error", err.Error()))
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         conn.addr,
		MaxRetries:   -1,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		PoolTimeout:  500 * time.Millisecond,
	})
	defer client.Close()
	return fn(client)
}

func (conn *Conn) Ping(ctx context.Context) error {
	if conn == nil || conn.addr == "" {
		return fmt.Errorf("valkey address is empty")
	}
	if err := probeTCP(conn.addr); err != nil {
		conn.disabled.Store(true)
		slog.Warn("valkey connection failed", slog.String("service", "sales"), slog.String("component", "valkey"), slog.String("addr", conn.addr), slog.String("error", err.Error()))
		return err
	}
	slog.Info("valkey connection successful", slog.String("service", "sales"), slog.String("component", "valkey"), slog.String("addr", conn.addr))
	return nil
}

func probeTCP(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
	if err != nil {
		return err
	}
	return conn.Close()
}
