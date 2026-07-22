package redis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "pending_reg:"
const otpKeyPrefix = "otp:"
const voiceOTPWaitPrefix = "voice:otp_wait:"
const voiceVerifiedPrefix = "voice:verified:"
const voiceOTPFailPrefix = "voice:otp_fail:"

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

type otpEntry struct {
	Token string `json:"token"`
	Code  string `json:"code"`
}

func normalizePhone(phone string) string {
	return sharedauth.CanonicalPhoneDigits(phone)
}

func (r *PendingRegistrationRepository) SetOTP(ctx context.Context, phone string, token string, code string, ttl time.Duration) error {
	key := otpKeyPrefix + normalizePhone(phone)
	b, err := json.Marshal(otpEntry{Token: token, Code: code})
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, b, ttl).Err()
}

func (r *PendingRegistrationRepository) GetOTP(ctx context.Context, phone string) (token string, code string, err error) {
	key := otpKeyPrefix + normalizePhone(phone)
	b, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return "", "", err
	}
	var e otpEntry
	if err := json.Unmarshal(b, &e); err != nil {
		return "", "", err
	}
	return e.Token, e.Code, nil
}

func (r *PendingRegistrationRepository) DeleteOTP(ctx context.Context, phone string) error {
	return r.client.Del(ctx, otpKeyPrefix+normalizePhone(phone)).Err()
}

func (r *PendingRegistrationRepository) SetVoiceOTPWait(ctx context.Context, phone string, voiceSessionID string, ttl time.Duration) error {
	return r.client.Set(ctx, voiceOTPWaitPrefix+normalizePhone(phone), voiceSessionID, ttl).Err()
}

func (r *PendingRegistrationRepository) GetVoiceOTPWait(ctx context.Context, phone string) (string, error) {
	return r.client.Get(ctx, voiceOTPWaitPrefix+normalizePhone(phone)).Result()
}

func (r *PendingRegistrationRepository) DeleteVoiceOTPWait(ctx context.Context, phone string) error {
	return r.client.Del(ctx, voiceOTPWaitPrefix+normalizePhone(phone)).Err()
}

func (r *PendingRegistrationRepository) SetVoiceVerified(ctx context.Context, voiceSessionID string, patientID string, ttl time.Duration) error {
	return r.client.Set(ctx, voiceVerifiedPrefix+voiceSessionID, patientID, ttl).Err()
}

func (r *PendingRegistrationRepository) ConsumeVoiceVerified(ctx context.Context, voiceSessionID string) (string, error) {
	key := voiceVerifiedPrefix + voiceSessionID
	val, err := r.client.GetDel(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (r *PendingRegistrationRepository) IncrVoiceOTPFail(ctx context.Context, phone string, window time.Duration) (int64, error) {
	key := voiceOTPFailPrefix + normalizePhone(phone)
	n, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		_ = r.client.Expire(ctx, key, window).Err()
	}
	return n, nil
}
