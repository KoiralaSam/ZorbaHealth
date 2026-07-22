package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *UserRepository) RegisterHospitalWithStaff(ctx context.Context, req domain.HospitalRegistration) (*domain.HospitalStaffAccount, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, phone_number, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.StaffEmail, nullableText(req.StaffPhone), req.PasswordHash, "hospital_staff").Scan(&userID); err != nil {
		return nil, err
	}

	var hospitalID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO hospitals (name, license_no, active)
		VALUES ($1, $2, true)
		RETURNING id
	`, req.HospitalName, req.LicenseNo).Scan(&hospitalID); err != nil {
		return nil, err
	}

	var staffID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO hospital_staff (hospital_id, user_id, email, password_hash, name, role, active, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7)
		RETURNING id
	`, hospitalID, userID, req.StaffEmail, req.PasswordHash, req.StaffName, req.StaffRole, nullableText(req.StaffPhone)).Scan(&staffID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &domain.HospitalStaffAccount{
		UserID:     userID.String(),
		HospitalID: hospitalID.String(),
		StaffID:    staffID.String(),
		StaffRole:  req.StaffRole,
	}, nil
}

func (r *UserRepository) RegisterStaff(ctx context.Context, req domain.HospitalStaffRegistration) (*domain.HospitalStaffAccount, error) {
	hospitalID, err := uuid.Parse(req.HospitalID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var active bool
	if err := tx.QueryRow(ctx, `SELECT active FROM hospitals WHERE id = $1`, hospitalID).Scan(&active); err != nil {
		return nil, err
	}
	if !active {
		return nil, fmt.Errorf("hospital is inactive")
	}

	var userID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, phone_number, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.StaffEmail, nullableText(req.StaffPhone), req.PasswordHash, "hospital_staff").Scan(&userID); err != nil {
		return nil, err
	}

	var staffID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO hospital_staff (hospital_id, user_id, email, password_hash, name, role, active, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7)
		RETURNING id
	`, hospitalID, userID, req.StaffEmail, req.PasswordHash, req.StaffName, req.StaffRole, nullableText(req.StaffPhone)).Scan(&staffID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &domain.HospitalStaffAccount{
		UserID:     userID.String(),
		HospitalID: hospitalID.String(),
		StaffID:    staffID.String(),
		StaffRole:  req.StaffRole,
	}, nil
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

func (r *UserRepository) ResolveSessionActor(ctx context.Context, userID, userRole, sessionID string) (*domain.SessionActor, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	pgUID := pgtype.UUID{Bytes: uid, Valid: true}

	if userRole == "patient" {
		patient, err := r.queries.GetPatientByUserID(ctx, pgUID)
		if err != nil {
			return nil, err
		}

		return &domain.SessionActor{
			ActorType: "patient",
			PatientID: uuid.UUID(patient.ID.Bytes).String(),
			SessionID: sessionID,
			Scopes:    []string{},
		}, nil
	}

	admin, err := r.queries.GetAdminByUserID(ctx, pgUID)
	if err == nil {
		return &domain.SessionActor{
			ActorType: "admin",
			AdminID:   uuid.UUID(admin.ID.Bytes).String(),
			Scopes:    []string{},
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	staff, err := r.queries.GetHospitalStaffByUserID(ctx, pgUID)
	if err == nil {
		return &domain.SessionActor{
			ActorType:  "staff",
			StaffID:    uuid.UUID(staff.ID.Bytes).String(),
			HospitalID: uuid.UUID(staff.HospitalID.Bytes).String(),
			Role:       staff.Role,
			Scopes:     []string{},
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return nil, fmt.Errorf("unable to resolve actor context for user_id=%s role=%s", userID, userRole)
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

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
