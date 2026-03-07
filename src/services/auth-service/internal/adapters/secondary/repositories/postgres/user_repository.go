package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/secondary/repositories/postgres/sqlc"
	domain "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/ports/outbound"
)

type UserRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewUserRepository(db *pgxpool.Pool) outbound.UserRepository {
	return &UserRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	params := sqlc.CreateUserParams{
		Email:        pgtype.Text{String: user.Email, Valid: user.Email != ""},
		PhoneNumber:  pgtype.Text{String: user.PhoneNumber, Valid: user.PhoneNumber != ""},
		PasswordHash: pgtype.Text{String: user.PasswordHash, Valid: user.PasswordHash != ""},
		Role:         user.Role,
	}

	dbUser, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.toDomainUser(&dbUser), nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	dbUser, err := r.queries.GetUserByID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, err
	}

	return r.toDomainUser(&dbUser), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, pgtype.Text{String: email, Valid: email != ""})
	if err != nil {
		return nil, err
	}

	return r.toDomainUser(&dbUser), nil
}

func (r *UserRepository) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByPhoneNumber(ctx, pgtype.Text{String: phoneNumber, Valid: phoneNumber != ""})
	if err != nil {
		return nil, err
	}

	return r.toDomainUser(&dbUser), nil
}

func (r *UserRepository) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	_, err = r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           pgtype.UUID{Bytes: uid, Valid: true},
		PasswordHash: pgtype.Text{String: passwordHash, Valid: passwordHash != ""},
	})
	return err
}

func (r *UserRepository) ListUsersByRole(ctx context.Context, role string, limit, offset int32) ([]*domain.User, error) {
	dbUsers, err := r.queries.ListUsersByRole(ctx, sqlc.ListUsersByRoleParams{
		Role:   role,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*domain.User, 0, len(dbUsers))
	for i := range dbUsers {
		out = append(out, r.toDomainUser(&dbUsers[i]))
	}
	return out, nil
}

func (r *UserRepository) toDomainUser(u *sqlc.User) *domain.User {
	var createdAt time.Time
	if u.CreatedAt.Valid {
		createdAt = u.CreatedAt.Time
	}

	return &domain.User{
		ID:           uuid.UUID(u.ID.Bytes).String(),
		Email:        u.Email.String,
		PhoneNumber:  u.PhoneNumber.String,
		PasswordHash: u.PasswordHash.String,
		Role:         u.Role,
		CreatedAt:    createdAt,
		UpdatedAt:    time.Time{}, // users table has no updated_at in sqlc model
	}
}

func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.DeleteUser(ctx, pgtype.UUID{Bytes: uid, Valid: true})
}