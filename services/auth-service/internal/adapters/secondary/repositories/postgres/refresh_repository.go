package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/secondary/repositories/postgres/sqlc"
	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/ports/outbound"
)

type RefreshRepository struct {
	db *pgxpool.Pool
}

func NewRefreshRepository(db *pgxpool.Pool) outbound.RefreshRepository {
	return &RefreshRepository{db: db}
}

func (r *RefreshRepository) GetAuthUUIDByTokenHash(ctx context.Context, tokenHash []byte) (string, error) {
	q := sqlc.New(r.db)
	row, err := q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domainerrors.ErrInvalidRefreshToken
		}
		return "", err
	}
	return uuid.UUID(row.AuthUuid.Bytes).String(), nil
}

func (r *RefreshRepository) InsertRefresh(ctx context.Context, authUUID, userID, actorType string, tokenHash []byte, generation int, expiresAt time.Time) error {
	authUID, err := uuid.Parse(authUUID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	q := sqlc.New(r.db)
	_, err = q.InsertRefreshToken(ctx, sqlc.InsertRefreshTokenParams{
		AuthUuid:   pgtype.UUID{Bytes: authUID, Valid: true},
		UserID:     pgtype.UUID{Bytes: uid, Valid: true},
		ActorType:  actorType,
		TokenHash:  tokenHash,
		Generation: int32(generation),
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	return err
}

func (r *RefreshRepository) RotateRefresh(ctx context.Context, presentedHash, newHash []byte, newExpiresAt time.Time) (int, string, string, string, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, "", "", "", false, err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)

	row, err := q.GetRefreshTokenByHashForUpdate(ctx, presentedHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", "", "", false, domainerrors.ErrInvalidRefreshToken
		}
		return 0, "", "", "", false, err
	}

	authUUID := uuid.UUID(row.AuthUuid.Bytes).String()
	userID := uuid.UUID(row.UserID.Bytes).String()
	actorType := row.ActorType

	authRow, err := q.GetAuthByAuthUUID(ctx, row.AuthUuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", "", "", false, domainerrors.ErrInvalidRefreshToken
		}
		return 0, "", "", "", false, err
	}
	if authRow.RevokedAt.Valid {
		return 0, "", "", "", false, domainerrors.ErrSessionRevoked
	}

	now := time.Now()
	if row.ExpiresAt.Valid && row.ExpiresAt.Time.Before(now) {
		return 0, "", "", "", false, domainerrors.ErrInvalidRefreshToken
	}

	if row.UsedAt.Valid || row.RevokedAt.Valid {
		_ = q.RevokeAuthByAuthUUID(ctx, sqlc.RevokeAuthByAuthUUIDParams{
			AuthUuid:      row.AuthUuid,
			RevokedReason: pgtype.Text{String: "reuse_detected", Valid: true},
		})
		_ = q.RevokeAllRefreshTokensForAuthUUID(ctx, row.AuthUuid)
		if err := tx.Commit(ctx); err != nil {
			return 0, "", "", "", false, err
		}
		return 0, authUUID, userID, actorType, true, domainerrors.ErrRefreshTokenReuse
	}

	if err := q.MarkRefreshTokenUsed(ctx, row.ID); err != nil {
		return 0, "", "", "", false, err
	}

	newGen := int(row.Generation) + 1
	_, err = q.InsertRefreshToken(ctx, sqlc.InsertRefreshTokenParams{
		AuthUuid:   row.AuthUuid,
		UserID:     row.UserID,
		ActorType:  actorType,
		TokenHash:  newHash,
		Generation: int32(newGen),
		ExpiresAt:  pgtype.Timestamptz{Time: newExpiresAt, Valid: true},
	})
	if err != nil {
		return 0, "", "", "", false, err
	}

	if err := q.TouchAuthByAuthUUID(ctx, row.AuthUuid); err != nil {
		return 0, "", "", "", false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, "", "", "", false, err
	}
	return newGen, authUUID, userID, actorType, false, nil
}

func (r *RefreshRepository) RevokeFamily(ctx context.Context, authUUID, reason string) error {
	authUID, err := uuid.Parse(authUUID)
	if err != nil {
		return err
	}
	q := sqlc.New(r.db)
	if err := q.RevokeAuthByAuthUUID(ctx, sqlc.RevokeAuthByAuthUUIDParams{
		AuthUuid:      pgtype.UUID{Bytes: authUID, Valid: true},
		RevokedReason: pgtype.Text{String: reason, Valid: true},
	}); err != nil {
		return err
	}
	return q.RevokeAllRefreshTokensForAuthUUID(ctx, pgtype.UUID{Bytes: authUID, Valid: true})
}
