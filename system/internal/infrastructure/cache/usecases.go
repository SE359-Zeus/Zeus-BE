package cache

import (
	"context"
	"time"

	"github.com/valkey-io/valkey-go"
)

func (v *valkeyConn) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(value).Ex(ttl).Build()).Error()
}

func (v *valkeyConn) Get(ctx context.Context, key string) (string, error) {
	val, err := v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if err == valkey.Nil {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

func (v *valkeyConn) Del(ctx context.Context, keys ...string) error {
	return v.client.Do(ctx, v.client.B().Del().Key(keys...).Build()).Error()
}

func (v *valkeyConn) SAdd(ctx context.Context, key, member string) error {
	return v.client.Do(ctx, v.client.B().Sadd().Key(key).Member(member).Build()).Error()
}

func (v *valkeyConn) SMembers(ctx context.Context, key string) ([]string, error) {
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

func (v *valkeyConn) Exists(ctx context.Context, key string) (bool, error) {
	n, err := v.client.Do(ctx, v.client.B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (v *valkeyConn) HSet(ctx context.Context, key, field, value string) error {
	return v.client.Do(ctx, v.client.B().Hset().Key(key).FieldValue().FieldValue(field, value).Build()).Error()
}

func (v *valkeyConn) HGetAll(ctx context.Context, key string) (map[string]string, error) {
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

func (v *valkeyConn) SIsMember(ctx context.Context, key, member string) (bool, error) {
	ok, err := v.client.Do(ctx, v.client.B().Sismember().Key(key).Member(member).Build()).AsBool()
	if err != nil {
		if err == valkey.Nil {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (v *valkeyConn) Close() {
	v.client.Close()
}
