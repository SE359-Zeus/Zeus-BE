package valkey

import (
	"context"
	"encoding/json"

	"zeus-sales-service/internal/models"

	"github.com/redis/go-redis/v9"
)

func (repo *Repository) EnqueueOrder(ctx context.Context, entry models.AllocationQueueEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	score := float64(entry.IngestedAt.UnixMicro())
	member := entry.OrderID.String()
	return repo.withClient(ctx, func(clientConn *redis.Client) error {
		pipeline := clientConn.TxPipeline()
		pipeline.HSet(ctx, repo.payloadKey, member, payload)
		pipeline.ZAdd(ctx, repo.queueKey, redis.Z{Score: score, Member: member})
		_, err = pipeline.Exec(ctx)
		return err
	})
}

func (repo *Repository) DequeueOrder(ctx context.Context) (*models.AllocationQueueEntry, error) {
	var entry *models.AllocationQueueEntry
	err := repo.withClient(ctx, func(clientConn *redis.Client) error {
		result, err := clientConn.ZPopMin(ctx, repo.queueKey, 1).Result()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		if len(result) == 0 {
			return nil
		}
		member := result[0].Member.(string)
		payload, err := clientConn.HGet(ctx, repo.payloadKey, member).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return err
		}
		if err := clientConn.HDel(ctx, repo.payloadKey, member).Err(); err != nil {
			return err
		}
		var decoded models.AllocationQueueEntry
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return err
		}
		entry = &decoded
		return nil
	})
	return entry, err
}

func (repo *Repository) ListQueue(ctx context.Context) ([]models.AllocationQueueEntry, error) {
	var entries []models.AllocationQueueEntry
	err := repo.withClient(ctx, func(clientConn *redis.Client) error {
		members, err := clientConn.ZRange(ctx, repo.queueKey, 0, -1).Result()
		if err != nil {
			return err
		}
		entries = make([]models.AllocationQueueEntry, 0, len(members))
		for _, member := range members {
			payload, err := clientConn.HGet(ctx, repo.payloadKey, member).Bytes()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				return err
			}
			var entry models.AllocationQueueEntry
			if err := json.Unmarshal(payload, &entry); err != nil {
				return err
			}
			entries = append(entries, entry)
		}
		return nil
	})
	return entries, err
}

func (repo *Repository) ClearQueue(ctx context.Context) error {
	return repo.withClient(ctx, func(clientConn *redis.Client) error {
		return clientConn.Del(ctx, repo.queueKey, repo.payloadKey).Err()
	})
}
