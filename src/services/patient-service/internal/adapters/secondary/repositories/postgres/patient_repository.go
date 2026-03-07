package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/postgres/sqlc"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
)

type PatientRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPatientRepository(db *pgxpool.Pool) outbound.PatientRepository {
	return &PatientRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// CreatePatient creates a new patient in the database
func (r *PatientRepository) CreatePatient(ctx context.Context, patient *models.Patient) (*models.Patient, error) {
	params := sqlc.CreatePatientParams{
		UserID:       pgtype.UUID{Bytes: patient.UserID, Valid: patient.UserID != uuid.Nil},
		PhoneNumber:  patient.PhoneNumber,
		Email:        pgtype.Text{String: patient.Email, Valid: patient.Email != ""},
		FullName:     pgtype.Text{String: patient.FullName, Valid: patient.FullName != ""},
		DateOfBirth:  pgtype.Date{Time: patient.DateOfBirth, Valid: !patient.DateOfBirth.IsZero()},
		MedicalNotes: pgtype.Text{String: patient.MedicalNotes, Valid: patient.MedicalNotes != ""},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	dbPatient, err := r.queries.CreatePatient(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.toDomainPatient(&dbPatient), nil
}

// GetPatientByID retrieves a patient by ID
func (r *PatientRepository) GetPatientByID(ctx context.Context, id string) (*models.Patient, error) {
	patientID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	dbPatient, err := r.queries.GetPatientByID(ctx, pgtype.UUID{Bytes: patientID, Valid: true})
	if err != nil {
		return nil, err
	}

	return r.toDomainPatient(&dbPatient), nil
}

// GetPatientByPhoneNumber retrieves a patient by phone number
func (r *PatientRepository) GetPatientByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Patient, error) {
	dbPatient, err := r.queries.GetPatientByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}

	return r.toDomainPatient(&dbPatient), nil
}

// GetPatientByEmail retrieves a patient by email
func (r *PatientRepository) GetPatientByEmail(ctx context.Context, email string) (*models.Patient, error) {
	dbPatient, err := r.queries.GetPatientByEmail(ctx, pgtype.Text{String: email, Valid: email != ""})
	if err != nil {
		return nil, err
	}

	return r.toDomainPatient(&dbPatient), nil
}

// UpdatePatient updates an existing patient
func (r *PatientRepository) UpdatePatient(ctx context.Context, patient *models.Patient) error {
	patientID, err := uuid.Parse(patient.ID.String())
	if err != nil {
		return err
	}

	params := sqlc.UpdatePatientParams{
		ID:           pgtype.UUID{Bytes: patientID, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Email:        pgtype.Text{String: patient.Email, Valid: patient.Email != ""},
		FullName:     pgtype.Text{String: patient.FullName, Valid: patient.FullName != ""},
		DateOfBirth:  pgtype.Date{Time: patient.DateOfBirth, Valid: !patient.DateOfBirth.IsZero()},
		MedicalNotes: pgtype.Text{String: patient.MedicalNotes, Valid: patient.MedicalNotes != ""},
	}

	_, err = r.queries.UpdatePatient(ctx, params)
	return err
}

// DeletePatient removes a patient from the database
func (r *PatientRepository) DeletePatient(ctx context.Context, id string) error {
	patientID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	return r.queries.DeletePatient(ctx, pgtype.UUID{Bytes: patientID, Valid: true})
}

// toDomainPatient converts sqlc.Patient to domain models.Patient
func (r *PatientRepository) toDomainPatient(p *sqlc.Patient) *models.Patient {
	var dateOfBirth, createdAt, updatedAt time.Time
	if p.DateOfBirth.Valid {
		dateOfBirth = p.DateOfBirth.Time
	}
	if p.CreatedAt.Valid {
		createdAt = p.CreatedAt.Time
	}
	if p.UpdatedAt.Valid {
		updatedAt = p.UpdatedAt.Time
	}

	return &models.Patient{
		ID:           uuid.UUID(p.ID.Bytes),
		UserID:       uuid.UUID(p.UserID.Bytes),
		PhoneNumber:  p.PhoneNumber,
		Email:        p.Email.String,
		FullName:     p.FullName.String,
		DateOfBirth:  dateOfBirth,
		MedicalNotes: p.MedicalNotes.String,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
