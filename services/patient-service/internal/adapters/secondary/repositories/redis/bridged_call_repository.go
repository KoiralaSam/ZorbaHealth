package redis

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedbridge "github.com/KoiralaSam/ZorbaHealth/shared/bridge"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/redis/go-redis/v9"
)

type BridgedCallRepository struct {
	client *redis.Client
}

func NewBridgedCallRepository() (outbound.BridgedCallRepository, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     env.GetString("REDIS_ADDR", "localhost:6379"),
		Password: env.GetString("REDIS_PASSWORD", ""),
		DB:       0,
	})
	if err := c.Ping(context.Background()).Err(); err != nil {
		log.Printf("failed to connect bridge redis: %v", err)
		return nil, err
	}
	return &BridgedCallRepository{client: c}, nil
}

func (r *BridgedCallRepository) Put(ctx context.Context, session *models.BridgedCallSession, ttl time.Duration) error {
	if session == nil {
		return nil
	}
	body, err := json.Marshal(toSharedSession(session))
	if err != nil {
		return err
	}
	return r.client.Set(ctx, sharedbridge.Key(strings.TrimSpace(session.SessionID)), body, ttl).Err()
}

func (r *BridgedCallRepository) Get(ctx context.Context, sessionID string) (*models.BridgedCallSession, error) {
	body, err := r.client.Get(ctx, sharedbridge.Key(strings.TrimSpace(sessionID))).Bytes()
	if err != nil {
		return nil, err
	}
	var session sharedbridge.Session
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, err
	}
	return fromSharedSession(&session), nil
}

// List scans bridged call sessions, returning those that match the hospital
// and optional status filter. Redis SCAN keeps this O(sessions) which is fine
// for the short-TTL session keyspace.
func (r *BridgedCallRepository) List(ctx context.Context, hospitalID string, status models.BridgedCallStatus, limit int) ([]*models.BridgedCallSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	hospitalID = strings.TrimSpace(hospitalID)

	var (
		out    []*models.BridgedCallSession
		cursor uint64
	)
	for {
		keys, next, err := r.client.Scan(ctx, cursor, sharedbridge.KeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			body, err := r.client.Get(ctx, key).Bytes()
			if err != nil {
				continue // expired between SCAN and GET
			}
			var session sharedbridge.Session
			if err := json.Unmarshal(body, &session); err != nil {
				continue
			}
			if hospitalID != "" && session.HospitalID != hospitalID {
				continue
			}
			if status != "" && session.Status != string(status) {
				continue
			}
			out = append(out, fromSharedSession(&session))
			if len(out) >= limit {
				return out, nil
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out, nil
}

func toSharedSession(s *models.BridgedCallSession) *sharedbridge.Session {
	return &sharedbridge.Session{
		SessionID:            s.SessionID,
		RoomSID:              s.RoomSID,
		PatientID:            s.PatientID,
		HospitalID:           s.HospitalID,
		StaffID:              s.StaffID,
		PatientAccessToken:   s.PatientAccessToken,
		StaffAccessToken:     s.StaffAccessToken,
		Status:               string(s.Status),
		RequestedByActorType: s.RequestedByActorType,
		RequestedByActorID:   s.RequestedByActorID,
		TransferReason:       s.TransferReason,
		RequestedAt:          s.RequestedAt,
		ConnectedAt:          s.ConnectedAt,
		EndedAt:              s.EndedAt,
		PatientTranslation:   toSharedPreferences(s.PatientTranslation),
		StaffTranslation:     toSharedPreferences(s.StaffTranslation),
	}
}

func fromSharedSession(s *sharedbridge.Session) *models.BridgedCallSession {
	return &models.BridgedCallSession{
		SessionID:            s.SessionID,
		RoomSID:              s.RoomSID,
		PatientID:            s.PatientID,
		HospitalID:           s.HospitalID,
		StaffID:              s.StaffID,
		PatientAccessToken:   s.PatientAccessToken,
		StaffAccessToken:     s.StaffAccessToken,
		Status:               models.BridgedCallStatus(s.Status),
		RequestedByActorType: s.RequestedByActorType,
		RequestedByActorID:   s.RequestedByActorID,
		TransferReason:       s.TransferReason,
		RequestedAt:          s.RequestedAt,
		ConnectedAt:          s.ConnectedAt,
		EndedAt:              s.EndedAt,
		PatientTranslation:   fromSharedPreferences(s.PatientTranslation),
		StaffTranslation:     fromSharedPreferences(s.StaffTranslation),
	}
}

func toSharedPreferences(p models.BridgedCallTranslationPreferences) sharedbridge.TranslationPreferences {
	return sharedbridge.TranslationPreferences{
		Enabled:             p.Enabled,
		LanguageMode:        string(p.LanguageMode),
		LanguageCode:        p.LanguageCode,
		ParticipantIdentity: p.ParticipantIdentity,
		UpdatedAt:           p.UpdatedAt,
	}
}

func fromSharedPreferences(p sharedbridge.TranslationPreferences) models.BridgedCallTranslationPreferences {
	return models.BridgedCallTranslationPreferences{
		Enabled:             p.Enabled,
		LanguageMode:        models.TranslationMode(p.LanguageMode),
		LanguageCode:        p.LanguageCode,
		ParticipantIdentity: p.ParticipantIdentity,
		UpdatedAt:           p.UpdatedAt,
	}
}
