package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/secondary/repositories/postgres/sqlc"
	domain "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/ports/outbound"
)

type AuthRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewAuthRepository(db *pgxpool.Pool) outbound.AuthRepository {
	return &AuthRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *AuthRepository) CreateAuth(ctx context.Context, userID, authUUID string) (*domain.Auth, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	dbAuth, err := r.queries.CreateAuth(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, err
	}

	return r.toDomainAuth(&dbAuth), nil
}

func (r *AuthRepository) GetAuthByUserIDAndAuthUUID(ctx context.Context, userID, authUUID string) (*domain.Auth, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	authUID, err := uuid.Parse(authUUID)
	if err != nil {
		return nil, err
	}

	dbAuth, err := r.queries.GetAuthByUserIDAndAuthUUID(ctx, sqlc.GetAuthByUserIDAndAuthUUIDParams{
		UserID:   pgtype.UUID{Bytes: uid, Valid: true},
		AuthUuid: pgtype.UUID{Bytes: authUID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return r.toDomainAuth(&dbAuth), nil
}

func (r *AuthRepository) DeleteAuth(ctx context.Context, userID, authUUID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	authUID, err := uuid.Parse(authUUID)
	if err != nil {
		return err
	}

	return r.queries.DeleteAuthByUserIDAndAuthUUID(ctx, sqlc.DeleteAuthByUserIDAndAuthUUIDParams{
		UserID:   pgtype.UUID{Bytes: uid, Valid: true},
		AuthUuid: pgtype.UUID{Bytes: authUID, Valid: true},
	})
}

func (r *AuthRepository) toDomainAuth(a *sqlc.Auth) *domain.Auth {
	return &domain.Auth{
		ID:       uint64(a.ID),
		UserID:   uuid.UUID(a.UserID.Bytes).String(),
		AuthUUID: uuid.UUID(a.AuthUuid.Bytes).String(),
	}
}
