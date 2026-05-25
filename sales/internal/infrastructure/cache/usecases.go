package cache

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	addr string
	ttl  time.Duration
}

func NewStore(addr string) *Store {
	return &Store{addr: strings.TrimSpace(addr), ttl: 30 * time.Minute}
}

func (store *Store) SetClient(ctx context.Context, client models.Client) error {
	if store == nil || store.addr == "" {
		return nil
	}
	payload, err := json.Marshal(client)
	if err != nil {
		return err
	}
	return store.withClient(ctx, func(clientConn *redis.Client) error {
		pipe := clientConn.TxPipeline()
		pipe.Set(ctx, store.clientKeyByID(client.ID), payload, store.ttl)
		pipe.Set(ctx, store.clientKeyByName(client.Name), payload, store.ttl)
		pipe.Del(ctx, store.clientsListKey())
		_, err = pipe.Exec(ctx)
		return err
	})
}

func (store *Store) GetClientByID(ctx context.Context, id uuid.UUID) (*models.Client, bool, error) {
	if store == nil || store.addr == "" || id == uuid.Nil {
		return nil, false, nil
	}
	var result *models.Client
	var ok bool
	err := store.withClient(ctx, func(clientConn *redis.Client) error {
		value, err := clientConn.Get(ctx, store.clientKeyByID(id)).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		var client models.Client
		if err := json.Unmarshal(value, &client); err != nil {
			return err
		}
		result = &client
		ok = true
		return nil
	})
	return result, ok, err
}

func (store *Store) GetClientByName(ctx context.Context, name string) (*models.Client, bool, error) {
	if store == nil || store.addr == "" {
		return nil, false, nil
	}
	var result *models.Client
	var ok bool
	err := store.withClient(ctx, func(clientConn *redis.Client) error {
		value, err := clientConn.Get(ctx, store.clientKeyByName(name)).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		var client models.Client
		if err := json.Unmarshal(value, &client); err != nil {
			return err
		}
		result = &client
		ok = true
		return nil
	})
	return result, ok, err
}

func (store *Store) GetClients(ctx context.Context) ([]models.Client, bool, error) {
	if store == nil || store.addr == "" {
		return nil, false, nil
	}
	var result []models.Client
	var ok bool
	err := store.withClient(ctx, func(clientConn *redis.Client) error {
		value, err := clientConn.Get(ctx, store.clientsListKey()).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		if err := json.Unmarshal(value, &result); err != nil {
			return err
		}
		ok = true
		return nil
	})
	return result, ok, err
}

func (store *Store) SetClients(ctx context.Context, clients []models.Client) error {
	if store == nil || store.addr == "" {
		return nil
	}
	payload, err := json.Marshal(clients)
	if err != nil {
		return err
	}
	return store.withClient(ctx, func(clientConn *redis.Client) error {
		return clientConn.Set(ctx, store.clientsListKey(), payload, store.ttl).Err()
	})
}

func (store *Store) DeleteClient(ctx context.Context, id uuid.UUID, name string) error {
	if store == nil || store.addr == "" {
		return nil
	}
	return store.withClient(ctx, func(clientConn *redis.Client) error {
		return clientConn.Del(ctx, store.clientKeyByID(id), store.clientKeyByName(name), store.clientsListKey()).Err()
	})
}

func (store *Store) SetStatus(ctx context.Context, status models.SalesOrderStatusLUT) error {
	if store == nil || store.addr == "" {
		return nil
	}
	payload, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return store.withClient(ctx, func(clientConn *redis.Client) error {
		pipe := clientConn.TxPipeline()
		pipe.Set(ctx, store.statusKeyByID(status.ID), payload, 0)
		pipe.Set(ctx, store.statusKeyByCode(status.Code), payload, 0)
		_, err = pipe.Exec(ctx)
		return err
	})
}

func (store *Store) GetStatusByID(ctx context.Context, id uuid.UUID) (*models.SalesOrderStatusLUT, bool, error) {
	if store == nil || store.addr == "" || id == uuid.Nil {
		return nil, false, nil
	}
	var result *models.SalesOrderStatusLUT
	var ok bool
	err := store.withClient(ctx, func(clientConn *redis.Client) error {
		value, err := clientConn.Get(ctx, store.statusKeyByID(id)).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		var status models.SalesOrderStatusLUT
		if err := json.Unmarshal(value, &status); err != nil {
			return err
		}
		result = &status
		ok = true
		return nil
	})
	return result, ok, err
}

func (store *Store) GetStatusByCode(ctx context.Context, code string) (*models.SalesOrderStatusLUT, bool, error) {
	if store == nil || store.addr == "" {
		return nil, false, nil
	}
	var result *models.SalesOrderStatusLUT
	var ok bool
	err := store.withClient(ctx, func(clientConn *redis.Client) error {
		value, err := clientConn.Get(ctx, store.statusKeyByCode(code)).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		var status models.SalesOrderStatusLUT
		if err := json.Unmarshal(value, &status); err != nil {
			return err
		}
		result = &status
		ok = true
		return nil
	})
	return result, ok, err
}

func (store *Store) withClient(ctx context.Context, fn func(*redis.Client) error) error {
	if store == nil || store.addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         store.addr,
		MaxRetries:   -1,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		PoolTimeout:  500 * time.Millisecond,
	})
	defer client.Close()
	return fn(client)
}

func (store *Store) clientKeyByID(id uuid.UUID) string {
	return "sales:cache:client:id:" + id.String()
}

func (store *Store) clientKeyByName(name string) string {
	return "sales:cache:client:name:" + strings.ToLower(strings.TrimSpace(name))
}

func (store *Store) clientsListKey() string {
	return "sales:cache:clients:list"
}

func (store *Store) statusKeyByID(id uuid.UUID) string {
	return "sales:cache:status:id:" + id.String()
}

func (store *Store) statusKeyByCode(code string) string {
	return "sales:cache:status:code:" + strings.ToUpper(strings.TrimSpace(code))
}
