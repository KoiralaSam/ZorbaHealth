package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/jackc/pgx/v5"
)

var defaultHospitalConsentPermissions = []string{
	"HEALTH_RECORD_ACCESS",
	"AI_SUMMARIZATION",
	"SCHEDULING",
}

func HospitalCreateConsentRequestHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	pool := db.GetDB()
	if pool == nil {
		writeJson(w, http.StatusServiceUnavailable, nil, &contracts.APIError{Code: "SERVICE_UNAVAILABLE", Message: "Database is not configured"})
		return
	}

	var body HospitalConsentRequestCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	permissions := normalizeHospitalConsentPermissions(body.RequestedPermissions)
	if len(permissions) == 0 {
		permissions = defaultHospitalConsentPermissions
	}
	expiresIn := body.ExpiresInMinutes
	if expiresIn <= 0 {
		expiresIn = 30
	}
	if expiresIn > 24*60 {
		expiresIn = 24 * 60
	}

	token, err := newConsentToken()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to generate consent token"})
		return
	}
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to encode permissions"})
		return
	}

	record, err := insertHospitalConsentRequest(r.Context(), token, claims.HospitalID, claims.StaffID, string(permissionsJSON), strings.TrimSpace(body.Note), time.Duration(expiresIn)*time.Minute)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to create consent request: " + err.Error()})
		return
	}

	writeJson(w, http.StatusCreated, HospitalConsentRequestCreateResponse{Request: record}, nil)
}

func HospitalListConsentRequestsHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	pool := db.GetDB()
	if pool == nil {
		writeJson(w, http.StatusServiceUnavailable, nil, &contracts.APIError{Code: "SERVICE_UNAVAILABLE", Message: "Database is not configured"})
		return
	}

	rows, err := pool.Query(r.Context(), `
		SELECT
			cr.id::text,
			cr.token,
			cr.hospital_id::text,
			h.name,
			cr.staff_id::text,
			hs.name,
				hs.role,
				COALESCE(cr.approved_patient_id::text, cr.patient_id::text, ''),
			cr.requested_permissions,
			COALESCE(cr.note, ''),
			cr.expires_at::text,
			COALESCE(cr.approved_at::text, ''),
			cr.created_at::text,
			CASE
				WHEN cr.approved_at IS NOT NULL THEN 'approved'
				WHEN cr.expires_at <= now() THEN 'expired'
				ELSE 'pending'
			END
		FROM hospital_consent_requests cr
		INNER JOIN hospitals h ON h.id = cr.hospital_id
		INNER JOIN hospital_staff hs ON hs.id = cr.staff_id
		WHERE cr.hospital_id = $1
		ORDER BY cr.created_at DESC
		LIMIT 50
	`, claims.HospitalID)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to list consent requests: " + err.Error()})
		return
	}
	defer rows.Close()

	requests := []HospitalConsentRequestRecord{}
	for rows.Next() {
		record, err := scanHospitalConsentRequest(rows)
		if err != nil {
			writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to read consent request: " + err.Error()})
			return
		}
		requests = append(requests, record)
	}
	if err := rows.Err(); err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to read consent requests: " + err.Error()})
		return
	}

	writeJson(w, http.StatusOK, HospitalConsentRequestListResponse{Requests: requests}, nil)
}

func PatientLookupConsentRequestHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Consent token is required"})
		return
	}

	record, err := lookupConsentRequest(r.Context(), token)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJson(w, http.StatusNotFound, nil, &contracts.APIError{Code: "NOT_FOUND", Message: "Consent request was not found"})
		return
	}
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to load consent request: " + err.Error()})
		return
	}
	if record.PatientID != "" && record.PatientID != claims.PatientID {
		writeJson(w, http.StatusForbidden, nil, &contracts.APIError{Code: "FORBIDDEN", Message: "This consent request is for a different patient"})
		return
	}
	if record.Status == "expired" {
		writeJson(w, http.StatusGone, nil, &contracts.APIError{Code: "EXPIRED", Message: "Consent request has expired"})
		return
	}

	writeJson(w, http.StatusOK, PatientConsentRequestLookupResponse{Request: record}, nil)
}

func PatientApproveConsentRequestHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	pool := db.GetDB()
	if pool == nil {
		writeJson(w, http.StatusServiceUnavailable, nil, &contracts.APIError{Code: "SERVICE_UNAVAILABLE", Message: "Database is not configured"})
		return
	}
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Consent token is required"})
		return
	}

	tx, err := pool.Begin(r.Context())
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to start consent transaction"})
		return
	}
	defer tx.Rollback(r.Context())

	var hospitalID, requestedPatientID, approvedAtText string
	var expired bool
	err = tx.QueryRow(r.Context(), `
		SELECT hospital_id::text, COALESCE(patient_id::text, ''), expires_at <= now(), COALESCE(approved_at::text, '')
		FROM hospital_consent_requests
		WHERE token = $1
		FOR UPDATE
	`, token).Scan(&hospitalID, &requestedPatientID, &expired, &approvedAtText)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJson(w, http.StatusNotFound, nil, &contracts.APIError{Code: "NOT_FOUND", Message: "Consent request was not found"})
		return
	}
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to load consent request: " + err.Error()})
		return
	}
	if approvedAtText != "" {
		writeJson(w, http.StatusConflict, nil, &contracts.APIError{Code: "ALREADY_APPROVED", Message: "Consent request has already been approved"})
		return
	}
	if expired {
		writeJson(w, http.StatusGone, nil, &contracts.APIError{Code: "EXPIRED", Message: "Consent request has expired"})
		return
	}
	if requestedPatientID != "" && requestedPatientID != claims.PatientID {
		writeJson(w, http.StatusForbidden, nil, &contracts.APIError{Code: "FORBIDDEN", Message: "This consent request is for a different patient"})
		return
	}

	var consent PatientHospitalConsentRecord
	err = tx.QueryRow(r.Context(), `
		INSERT INTO patient_hospital_consents (patient_id, hospital_id, granted_at, revoked_at)
		VALUES ($1, $2, now(), NULL)
		ON CONFLICT (patient_id, hospital_id)
		DO UPDATE SET granted_at = now(), revoked_at = NULL
		RETURNING hospital_id::text, granted_at::text, COALESCE(revoked_at::text, '')
	`, claims.PatientID, hospitalID).Scan(&consent.HospitalID, &consent.GrantedAt, &consent.RevokedAt)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to grant hospital consent: " + err.Error()})
		return
	}

	err = tx.QueryRow(r.Context(), `SELECT name FROM hospitals WHERE id = $1`, hospitalID).Scan(&consent.HospitalName)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to load hospital name: " + err.Error()})
		return
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE hospital_consent_requests
		SET approved_at = now(), approved_patient_id = $2
		WHERE token = $1
	`, token, claims.PatientID); err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to mark consent request approved: " + err.Error()})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to commit consent approval"})
		return
	}

	consent.Status = "active"
	writeJson(w, http.StatusOK, PatientConsentRequestApproveResponse{Message: "Hospital consent granted.", Consent: consent}, nil)
}

func PatientListHospitalConsentsHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	pool := db.GetDB()
	if pool == nil {
		writeJson(w, http.StatusServiceUnavailable, nil, &contracts.APIError{Code: "SERVICE_UNAVAILABLE", Message: "Database is not configured"})
		return
	}

	rows, err := pool.Query(r.Context(), `
		SELECT c.hospital_id::text, h.name, c.granted_at::text, COALESCE(c.revoked_at::text, '')
		FROM patient_hospital_consents c
		INNER JOIN hospitals h ON h.id = c.hospital_id
		WHERE c.patient_id = $1
		ORDER BY c.granted_at DESC
	`, claims.PatientID)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to list hospital consents: " + err.Error()})
		return
	}
	defer rows.Close()

	consents := []PatientHospitalConsentRecord{}
	for rows.Next() {
		var consent PatientHospitalConsentRecord
		if err := rows.Scan(&consent.HospitalID, &consent.HospitalName, &consent.GrantedAt, &consent.RevokedAt); err != nil {
			writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to read hospital consent: " + err.Error()})
			return
		}
		consent.Status = "active"
		if consent.RevokedAt != "" {
			consent.Status = "revoked"
		}
		consents = append(consents, consent)
	}
	if err := rows.Err(); err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to read hospital consents: " + err.Error()})
		return
	}

	writeJson(w, http.StatusOK, PatientHospitalConsentListResponse{Consents: consents}, nil)
}

func PatientRevokeHospitalConsentHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	pool := db.GetDB()
	if pool == nil {
		writeJson(w, http.StatusServiceUnavailable, nil, &contracts.APIError{Code: "SERVICE_UNAVAILABLE", Message: "Database is not configured"})
		return
	}
	hospitalID := strings.TrimSpace(r.PathValue("hospital_id"))
	if hospitalID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "hospital_id is required"})
		return
	}
	tag, err := pool.Exec(r.Context(), `
		UPDATE patient_hospital_consents
		SET revoked_at = now()
		WHERE patient_id = $1 AND hospital_id = $2 AND revoked_at IS NULL
	`, claims.PatientID, hospitalID)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to revoke hospital consent: " + err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJson(w, http.StatusNotFound, nil, &contracts.APIError{Code: "NOT_FOUND", Message: "Active hospital consent was not found"})
		return
	}

	writeJson(w, http.StatusOK, map[string]string{"message": "Hospital consent revoked."}, nil)
}

func insertHospitalConsentRequest(ctx context.Context, token, hospitalID, staffID, permissionsJSON, note string, ttl time.Duration) (HospitalConsentRequestRecord, error) {
	pool := db.GetDB()
	var noteSQL any
	if note != "" {
		noteSQL = note
	}
	var record HospitalConsentRequestRecord
	row := pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO hospital_consent_requests (token, hospital_id, staff_id, requested_permissions, note, expires_at)
			VALUES ($1, $2, $3, $4, $5, now() + ($6 * interval '1 second'))
			RETURNING *
		)
		SELECT
			inserted.id::text,
			inserted.token,
			inserted.hospital_id::text,
			h.name,
			inserted.staff_id::text,
			hs.name,
			hs.role,
				COALESCE(inserted.approved_patient_id::text, inserted.patient_id::text, ''),
			inserted.requested_permissions,
			COALESCE(inserted.note, ''),
			inserted.expires_at::text,
			COALESCE(inserted.approved_at::text, ''),
			inserted.created_at::text,
			'pending'
		FROM inserted
		INNER JOIN hospitals h ON h.id = inserted.hospital_id
		INNER JOIN hospital_staff hs ON hs.id = inserted.staff_id
	`, token, hospitalID, staffID, permissionsJSON, noteSQL, int(ttl.Seconds()))
	if err := scanHospitalConsentRequestRow(row, &record); err != nil {
		return HospitalConsentRequestRecord{}, err
	}
	return record, nil
}

func lookupConsentRequest(ctx context.Context, token string) (HospitalConsentRequestRecord, error) {
	pool := db.GetDB()
	if pool == nil {
		return HospitalConsentRequestRecord{}, errors.New("database is not configured")
	}
	var record HospitalConsentRequestRecord
	row := pool.QueryRow(ctx, `
		SELECT
			cr.id::text,
			cr.token,
			cr.hospital_id::text,
			h.name,
			cr.staff_id::text,
			hs.name,
			hs.role,
				COALESCE(cr.approved_patient_id::text, cr.patient_id::text, ''),
			cr.requested_permissions,
			COALESCE(cr.note, ''),
			cr.expires_at::text,
			COALESCE(cr.approved_at::text, ''),
			cr.created_at::text,
			CASE
				WHEN cr.approved_at IS NOT NULL THEN 'approved'
				WHEN cr.expires_at <= now() THEN 'expired'
				ELSE 'pending'
			END
		FROM hospital_consent_requests cr
		INNER JOIN hospitals h ON h.id = cr.hospital_id
		INNER JOIN hospital_staff hs ON hs.id = cr.staff_id
		WHERE cr.token = $1
	`, token)
	if err := scanHospitalConsentRequestRow(row, &record); err != nil {
		return HospitalConsentRequestRecord{}, err
	}
	return record, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHospitalConsentRequest(rows pgx.Rows) (HospitalConsentRequestRecord, error) {
	var record HospitalConsentRequestRecord
	if err := scanHospitalConsentRequestRow(rows, &record); err != nil {
		return HospitalConsentRequestRecord{}, err
	}
	return record, nil
}

func scanHospitalConsentRequestRow(row rowScanner, record *HospitalConsentRequestRecord) error {
	var permissionsJSON string
	if err := row.Scan(
		&record.ID,
		&record.Token,
		&record.HospitalID,
		&record.HospitalName,
		&record.StaffID,
		&record.StaffName,
		&record.StaffRole,
		&record.PatientID,
		&permissionsJSON,
		&record.Note,
		&record.ExpiresAt,
		&record.ApprovedAt,
		&record.CreatedAt,
		&record.Status,
	); err != nil {
		return err
	}
	record.RequestedPermissions = decodeConsentPermissions(permissionsJSON)
	record.QRPayload = consentQRPayload(record.Token)
	return nil
}

func newConsentToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func normalizeHospitalConsentPermissions(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToUpper(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func decodeConsentPermissions(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func consentQRPayload(token string) string {
	payload, _ := json.Marshal(map[string]any{
		"type":    "zorba_hospital_consent",
		"version": 1,
		"token":   token,
	})
	return string(payload)
}
