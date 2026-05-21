package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

type Valkey struct {
	client valkey.Client
}

func NewValkey(addr string) (*Valkey, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}

	if err := client.Do(context.Background(), client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Valkey: %w", err)
	}

	return &Valkey{client: client}, nil
}

func (v *Valkey) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(value).Ex(ttl).Build()).Error()
}

func (v *Valkey) Get(ctx context.Context, key string) (string, error) {
	val, err := v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if err == valkey.Nil {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

func (v *Valkey) Del(ctx context.Context, keys ...string) error {
	return v.client.Do(ctx, v.client.B().Del().Key(keys...).Build()).Error()
}

func (v *Valkey) SAdd(ctx context.Context, key, member string) error {
	return v.client.Do(ctx, v.client.B().Sadd().Key(key).Member(member).Build()).Error()
}

func (v *Valkey) SMembers(ctx context.Context, key string) ([]string, error) {
	resp := v.client.Do(ctx, v.client.B().Smembers().Key(key).Build())
	vals, err := resp.AsStrSlice()
	if err != nil {
		if err == valkey.Nil {
			return nil, nil
		}
		return nil, err
	}
	return vals, nil
}

func (v *Valkey) Exists(ctx context.Context, key string) (bool, error) {
	n, err := v.client.Do(ctx, v.client.B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (v *Valkey) HSet(ctx context.Context, key, field, value string) error {
	return v.client.Do(ctx, v.client.B().Hset().Key(key).FieldValue().FieldValue(field, value).Build()).Error()
}

func (v *Valkey) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	resp := v.client.Do(ctx, v.client.B().Hgetall().Key(key).Build())
	m, err := resp.AsStrMap()
	if err != nil {
		if err == valkey.Nil {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

func (v *Valkey) SIsMember(ctx context.Context, key, member string) (bool, error) {
	ok, err := v.client.Do(ctx, v.client.B().Sismember().Key(key).Member(member).Build()).AsBool()
	if err != nil {
		if err == valkey.Nil {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (v *Valkey) Close() {
	v.client.Close()
}
