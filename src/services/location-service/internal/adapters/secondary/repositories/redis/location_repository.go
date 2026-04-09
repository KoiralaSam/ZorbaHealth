package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	goredis "github.com/redis/go-redis/v9"
)

const defaultKeyPrefix = "location:session:"

var _ outbound.LocationRepository = (*LocationRepository)(nil)

// LocationRepository stores live session locations in Redis.
type LocationRepository struct {
	client    *goredis.Client
	keyPrefix string
	ttl       time.Duration
}

// NewLocationRepository connects to Redis using REDIS_ADDR / REDIS_PASSWORD.
// LOCATION_REDIS_DB selects the DB index (default 0).
// LOCATION_REDIS_KEY_PREFIX overrides the key prefix (default "location:session:").
// LOCATION_REDIS_SESSION_TTL_SEC sets key TTL; 0 means no expiry (keys removed on Delete or manual cleanup).
func NewLocationRepository() (outbound.LocationRepository, error) {
	db := env.GetInt("LOCATION_REDIS_DB", 0)
	ttlSec := env.GetInt("LOCATION_REDIS_SESSION_TTL_SEC", 0)
	prefix := env.GetString("LOCATION_REDIS_KEY_PREFIX", defaultKeyPrefix)

	c := goredis.NewClient(&goredis.Options{
		Addr:     env.GetString("REDIS_ADDR", "localhost:6379"),
		Password: env.GetString("REDIS_PASSWORD", ""),
		DB:       db,
	})
	if err := c.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	var ttl time.Duration
	if ttlSec > 0 {
		ttl = time.Duration(ttlSec) * time.Second
	}
	return &LocationRepository{
		client:    c,
		keyPrefix: prefix,
		ttl:       ttl,
	}, nil
}

func (r *LocationRepository) key(sessionID string) string {
	return r.keyPrefix + sessionID
}

func (r *LocationRepository) Save(ctx context.Context, sessionID string, loc models.Location) error {
	loc.CapturedAt = time.Now().UTC()
	b, err := json.Marshal(loc)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key(sessionID), b, r.ttl).Err()
}

func (r *LocationRepository) Get(ctx context.Context, sessionID string) (*models.Location, error) {
	b, err := r.client.Get(ctx, r.key(sessionID)).Bytes()
	if err == goredis.Nil {
		return nil, errors.ErrNoLocationFound
	}
	if err != nil {
		return nil, err
	}
	var out models.Location
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *LocationRepository) Delete(ctx context.Context, sessionID string) error {
	return r.client.Del(ctx, r.key(sessionID)).Err()
}
