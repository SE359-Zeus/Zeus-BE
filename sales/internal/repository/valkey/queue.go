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
	pipeline := repo.client.TxPipeline()
	pipeline.HSet(ctx, repo.payloadKey, member, payload)
	pipeline.ZAdd(ctx, repo.queueKey, redis.Z{Score: score, Member: member})
	_, err = pipeline.Exec(ctx)
	return err
}

func (repo *Repository) DequeueOrder(ctx context.Context) (*models.AllocationQueueEntry, error) {
	result, err := repo.client.ZPopMin(ctx, repo.queueKey, 1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	member := result[0].Member.(string)
	payload, err := repo.client.HGet(ctx, repo.payloadKey, member).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if err := repo.client.HDel(ctx, repo.payloadKey, member).Err(); err != nil {
		return nil, err
	}
	var entry models.AllocationQueueEntry
	if err := json.Unmarshal(payload, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (repo *Repository) ListQueue(ctx context.Context) ([]models.AllocationQueueEntry, error) {
	members, err := repo.client.ZRange(ctx, repo.queueKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	entries := make([]models.AllocationQueueEntry, 0, len(members))
	for _, member := range members {
		payload, err := repo.client.HGet(ctx, repo.payloadKey, member).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, err
		}
		var entry models.AllocationQueueEntry
		if err := json.Unmarshal(payload, &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (repo *Repository) ClearQueue(ctx context.Context) error {
	return repo.client.Del(ctx, repo.queueKey, repo.payloadKey).Err()
}
