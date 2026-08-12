package postgres

import (
	"context"
	"fmt"

	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/ports/outbound"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ outbound.HospitalRepository = (*HospitalRepository)(nil)

type HospitalRepository struct {
	db *pgxpool.Pool
}

func NewHospitalRepository(db *pgxpool.Pool) *HospitalRepository {
	return &HospitalRepository{db: db}
}

func (r *HospitalRepository) CreateHospitalWithStaff(ctx context.Context, in models.CreateHospitalStaffInput) (*models.HospitalStaffAccount, error) {
	userID, err := uuid.Parse(in.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var hospitalID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO hospitals (name, license_no, active)
		VALUES ($1, $2, true)
		RETURNING id
	`, in.HospitalName, in.LicenseNo).Scan(&hospitalID); err != nil {
		return nil, err
	}

	var staffID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO hospital_staff (hospital_id, user_id, email, password_hash, name, role, active, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7)
		RETURNING id
	`, hospitalID, userID, in.StaffEmail, "", in.StaffName, in.StaffRole, nullableText(in.StaffPhone)).Scan(&staffID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &models.HospitalStaffAccount{
		UserID:     userID.String(),
		HospitalID: hospitalID.String(),
		StaffID:    staffID.String(),
		StaffRole:  in.StaffRole,
	}, nil
}

func (r *HospitalRepository) CreateStaffForHospital(ctx context.Context, in models.CreateHospitalStaffInput) (*models.HospitalStaffAccount, error) {
	hospitalID, err := uuid.Parse(in.HospitalID)
	if err != nil {
		return nil, fmt.Errorf("invalid hospital_id: %w", err)
	}
	userID, err := uuid.Parse(in.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
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

	var staffID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO hospital_staff (hospital_id, user_id, email, password_hash, name, role, active, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7)
		RETURNING id
	`, hospitalID, userID, in.StaffEmail, "", in.StaffName, in.StaffRole, nullableText(in.StaffPhone)).Scan(&staffID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &models.HospitalStaffAccount{
		UserID:     userID.String(),
		HospitalID: hospitalID.String(),
		StaffID:    staffID.String(),
		StaffRole:  in.StaffRole,
	}, nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
