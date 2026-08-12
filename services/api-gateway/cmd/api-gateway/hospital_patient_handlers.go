package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/jackc/pgx/v5"
)

func HospitalPatientsHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	patients, apiErr := listHospitalPatients(r.Context(), claims.HospitalID, r.URL.Query().Get("query"), 60)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	writeJson(w, http.StatusOK, HospitalPatientListResponse{Patients: patients}, nil)
}

func resolveHospitalPatientLookup(ctx context.Context, hospitalID string, lookup string) (HospitalPatientRecord, *contracts.APIError) {
	patients, apiErr := listHospitalPatients(ctx, hospitalID, lookup, 2)
	if apiErr != nil {
		return HospitalPatientRecord{}, apiErr
	}
	if len(patients) == 0 {
		return HospitalPatientRecord{}, &contracts.APIError{
			Code:    "NOT_FOUND",
			Message: "No consented patient matched that name, email, or ID.",
		}
	}
	if len(patients) > 1 {
		return HospitalPatientRecord{}, &contracts.APIError{
			Code:    "AMBIGUOUS_PATIENT_LOOKUP",
			Message: "Multiple consented patients matched. Use a more specific name, email, or patient ID.",
		}
	}
	return patients[0], nil
}

func listHospitalPatients(ctx context.Context, hospitalID string, lookup string, limit int) ([]HospitalPatientRecord, *contracts.APIError) {
	pool := db.GetDB()
	if pool == nil {
		return nil, &contracts.APIError{Code: "SERVICE_UNAVAILABLE", Message: "Database is not configured"}
	}
	if limit <= 0 || limit > 100 {
		limit = 60
	}
	query := strings.TrimSpace(lookup)

	rows, err := pool.Query(ctx, `
		SELECT
			p.id::text,
			COALESCE(p.full_name, ''),
			COALESCE(p.email, ''),
			COALESCE(p.phone_number, ''),
			COALESCE(p.date_of_birth::text, ''),
			c.granted_at::text,
			COALESCE(last_call.last_call_at::text, '')
		FROM patient_hospital_consents c
		INNER JOIN patients p ON p.id = c.patient_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(calls.ended_at, calls.started_at) AS last_call_at
			FROM calls
			WHERE calls.patient_id = p.id
			ORDER BY COALESCE(calls.ended_at, calls.started_at) DESC NULLS LAST
			LIMIT 1
		) last_call ON true
		WHERE c.hospital_id = $1
			AND c.revoked_at IS NULL
			AND (
				$2 = ''
				OR p.id::text = $2
				OR lower(COALESCE(p.email, '')) = lower($2)
				OR lower(COALESCE(p.email, '')) LIKE lower('%' || $2 || '%')
				OR p.full_name ILIKE '%' || $2 || '%'
				OR (
					SELECT COALESCE(bool_and(p.full_name ILIKE '%' || trim(both from t) || '%'), false)
					FROM unnest(string_to_array(trim(both from $2), ' ')) AS t
					WHERE trim(both from t) <> ''
				)
			)
		ORDER BY
			CASE
				WHEN lower(COALESCE(p.full_name, '')) = lower($2) THEN 0
				WHEN p.full_name ILIKE $2 || '%' THEN 1
				WHEN p.full_name ILIKE '%' || $2 || '%' THEN 2
				ELSE 3
			END,
			c.granted_at DESC
		LIMIT $3
	`, hospitalID, query, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return []HospitalPatientRecord{}, nil
	}
	if err != nil {
		return nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to load hospital patients: " + err.Error()}
	}
	defer rows.Close()

	patients := []HospitalPatientRecord{}
	for rows.Next() {
		var patient HospitalPatientRecord
		if err := rows.Scan(
			&patient.PatientID,
			&patient.FullName,
			&patient.Email,
			&patient.PhoneNumber,
			&patient.DateOfBirth,
			&patient.ConsentGrantedAt,
			&patient.LastCallAt,
		); err != nil {
			return nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to read hospital patient: " + err.Error()}
		}
		patients = append(patients, patient)
	}
	if err := rows.Err(); err != nil {
		return nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to read hospital patients: " + err.Error()}
	}
	return patients, nil
}
