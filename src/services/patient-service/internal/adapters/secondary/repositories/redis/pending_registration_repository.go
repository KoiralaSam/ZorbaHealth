package redis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "pending_reg:"

type PendingRegistrationRepository struct {
	client *redis.Client
}

func NewPendingRegistrationRepository() (outbound.PendingRegistrationRepository, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     env.GetString("REDIS_ADDR", "localhost:6379"),
		Password: env.GetString("REDIS_PASSWORD", ""),
		DB:       0,
	})
	if err := c.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
		return nil, err
	}
	return &PendingRegistrationRepository{
		client: c,
	}, nil
}

func (r *PendingRegistrationRepository) Set(ctx context.Context, token string, data *models.PendingRegistration, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, keyPrefix+token, b, ttl).Err()
}

func (r *PendingRegistrationRepository) Get(ctx context.Context, token string) (*models.PendingRegistration, error) {
	b, err := r.client.Get(ctx, keyPrefix+token).Bytes()
	if err != nil {
		return nil, err
	}
	var out models.PendingRegistration
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PendingRegistrationRepository) Delete(ctx context.Context, token string) error {
	return r.client.Del(ctx, keyPrefix+token).Err()
}
