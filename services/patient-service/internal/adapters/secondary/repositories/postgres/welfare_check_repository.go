package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *PatientRepository) InsertWelfareCheck(ctx context.Context, check *models.WelfareCheck) (*models.WelfareCheck, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO welfare_check_requests (
			patient_id, scheduled_at, timezone, reason_code, reason_detail, status
		) VALUES ($1, $2, $3, $4, $5, 'scheduled')
		RETURNING id::text, patient_id::text, scheduled_at, timezone, reason_code, reason_detail,
		          status, COALESCE(recurrence_rule, ''), recurrence_starts_at, recurrence_ends_at,
		          created_at, updated_at, cancelled_at
	`, check.PatientID, check.ScheduledAt, check.Timezone, string(check.ReasonCode), check.ReasonDetail)

	saved, err := scanWelfareCheck(row)
	if err != nil {
		return nil, err
	}
	var runID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO welfare_check_runs (
			request_id, patient_id, scheduled_at, status, next_attempt_at
		) VALUES ($1, $2, $3, 'pending', $3)
		RETURNING id
	`, saved.ID, saved.PatientID, saved.ScheduledAt).Scan(&runID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE welfare_check_runs
		SET livekit_room_name = $2
		WHERE id = $1
	`, runID, stableWelfareRoomName(runID))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *PatientRepository) ListWelfareChecks(ctx context.Context, filter models.ListWelfareChecksFilter) ([]models.WelfareCheck, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			w.id::text,
			w.patient_id::text,
			w.scheduled_at,
			w.timezone,
			w.reason_code,
			w.reason_detail,
			w.status,
			COALESCE(w.recurrence_rule, ''),
			w.recurrence_starts_at,
			w.recurrence_ends_at,
			w.created_at,
			w.updated_at,
			w.cancelled_at,
			COALESCE(lr.id::text, '') AS latest_run_id,
			COALESCE(lr.status, '') AS latest_run_status,
			COALESCE(lr.attempts, 0)::int AS latest_run_attempts,
			COALESCE(lr.failure_reason, '') AS latest_run_failure_reason
		FROM welfare_check_requests w
		LEFT JOIN LATERAL (
			SELECT id, status, attempts, failure_reason
			FROM welfare_check_runs
			WHERE request_id = w.id
			ORDER BY created_at DESC
			LIMIT 1
		) lr ON true
		WHERE w.patient_id = $1
		  AND ($2::boolean OR w.status <> 'cancelled')
		ORDER BY w.scheduled_at DESC
		LIMIT $3
	`, filter.PatientID, filter.IncludeCancelled, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.WelfareCheck, 0)
	for rows.Next() {
		check, err := scanWelfareCheckWithRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *check)
	}
	return out, rows.Err()
}

func (r *PatientRepository) CancelWelfareCheck(ctx context.Context, patientID, checkID uuid.UUID) (*models.WelfareCheck, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		UPDATE welfare_check_requests
		SET status = 'cancelled', cancelled_at = now(), updated_at = now()
		WHERE id = $1
		  AND patient_id = $2
		  AND status = 'scheduled'
		  AND NOT EXISTS (
		    SELECT 1
		    FROM welfare_check_runs
		    WHERE request_id = $1
		      AND status NOT IN ('pending', 'claimed', 'cancelled')
		  )
		RETURNING id::text, patient_id::text, scheduled_at, timezone, reason_code, reason_detail,
		          status, COALESCE(recurrence_rule, ''), recurrence_starts_at, recurrence_ends_at,
		          created_at, updated_at, cancelled_at
	`, checkID, patientID)
	check, err := scanWelfareCheck(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrWelfareCheckNotFound
		}
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE welfare_check_runs
		SET status = 'cancelled', updated_at = now()
		WHERE request_id = $1
		  AND status IN ('pending', 'claimed')
	`, checkID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return check, nil
}

func (r *PatientRepository) ClaimDueWelfareCheckRuns(ctx context.Context, limit int32) ([]models.WelfareCheckRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH due AS (
			SELECT r.id
			FROM welfare_check_runs r
			INNER JOIN welfare_check_requests w ON w.id = r.request_id
			WHERE (
			        r.status = 'pending'
			     OR (r.status = 'claimed' AND r.updated_at < now() - interval '10 minutes')
			  )
			  AND w.status = 'scheduled'
			  AND r.scheduled_at <= now()
			  AND COALESCE(r.next_attempt_at, r.scheduled_at) <= now()
			ORDER BY r.scheduled_at ASC
			FOR UPDATE OF r SKIP LOCKED
			LIMIT $1
		)
		UPDATE welfare_check_runs r
		SET status = 'claimed',
		    attempts = attempts + 1,
		    last_attempt_at = now(),
		    updated_at = now(),
		    livekit_room_name = COALESCE(NULLIF(r.livekit_room_name, ''), 'welfare-check-' || r.id::text)
		FROM due, welfare_check_requests w, patients p
		WHERE r.id = due.id
		  AND w.id = r.request_id
		  AND p.id = r.patient_id
		RETURNING
			r.id::text,
			r.request_id::text,
			r.patient_id::text,
			r.scheduled_at,
			r.status,
			r.attempts,
			r.last_attempt_at,
			r.next_attempt_at,
			COALESCE(r.livekit_room_name, ''),
			COALESCE(r.livekit_room_sid, ''),
			COALESCE(r.livekit_dispatch_id, ''),
			COALESCE(r.livekit_sip_call_id, ''),
			COALESCE(r.failure_reason, ''),
			r.created_at,
			r.updated_at,
			w.reason_code,
			w.reason_detail,
			w.timezone,
			p.phone_number,
			COALESCE(p.full_name, ''),
			p.user_id::text
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.WelfareCheckRun, 0)
	for rows.Next() {
		run, err := scanWelfareCheckRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PatientRepository) PersistWelfareRunLiveKitResult(ctx context.Context, result models.WelfareCheckDispatchResult) error {
	_, err := r.db.Exec(ctx, `
		UPDATE welfare_check_runs
		SET livekit_room_name = $2,
		    livekit_room_sid = $3,
		    livekit_dispatch_id = $4,
		    livekit_sip_call_id = $5,
		    updated_at = now()
		WHERE id = $1
		  AND status IN ('claimed', 'dispatched')
	`, result.RunID, result.RoomName, result.RoomSID, result.DispatchID, result.SIPCallID)
	return err
}

func (r *PatientRepository) MarkWelfareRunDispatched(ctx context.Context, result models.WelfareCheckDispatchResult) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE welfare_check_runs
		SET status = 'dispatched',
		    livekit_room_name = $2,
		    livekit_room_sid = $3,
		    livekit_dispatch_id = $4,
		    livekit_sip_call_id = $5,
		    failure_reason = NULL,
		    updated_at = now()
		WHERE id = $1
		  AND status IN ('claimed', 'dispatched')
	`, result.RunID, result.RoomName, result.RoomSID, result.DispatchID, result.SIPCallID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainErrors.ErrWelfareCheckRunTransition
	}
	return nil
}

func (r *PatientRepository) MarkWelfareRunFailed(ctx context.Context, runID uuid.UUID, reason string, nextAttemptAt *time.Time) error {
	retry := nextAttemptAt != nil
	status := "failed"
	var nextAttempt any
	if retry {
		status = "pending"
		nextAttempt = nextAttemptAt.UTC()
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE welfare_check_runs
		SET status = $2,
		    failure_reason = $3,
		    next_attempt_at = $4,
		    livekit_room_sid = CASE WHEN $5 THEN '' ELSE livekit_room_sid END,
		    livekit_dispatch_id = CASE WHEN $5 THEN '' ELSE livekit_dispatch_id END,
		    livekit_sip_call_id = CASE WHEN $5 THEN '' ELSE livekit_sip_call_id END,
		    updated_at = now()
		WHERE id = $1
		  AND status IN ('claimed', 'pending', 'dispatched')
	`, runID, status, reason, nextAttempt, retry)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainErrors.ErrWelfareCheckRunTransition
	}
	if !retry {
		_, err = tx.Exec(ctx, `
			UPDATE welfare_check_requests w
			SET status = 'failed', updated_at = now()
			FROM welfare_check_runs r
			WHERE r.id = $1
			  AND w.id = r.request_id
			  AND w.status = 'scheduled'
		`, runID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PatientRepository) MarkWelfareRunMissed(ctx context.Context, runID uuid.UUID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE welfare_check_runs
		SET status = 'missed',
		    failure_reason = $2,
		    updated_at = now()
		WHERE id = $1
		  AND status IN ('claimed', 'pending', 'dispatched', 'answered')
	`, runID, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainErrors.ErrWelfareCheckRunTransition
	}
	_, err = tx.Exec(ctx, `
		UPDATE welfare_check_requests w
		SET status = 'missed', updated_at = now()
		FROM welfare_check_runs r
		WHERE r.id = $1
		  AND w.id = r.request_id
		  AND w.status = 'scheduled'
	`, runID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PatientRepository) GetWelfareCheckRun(ctx context.Context, patientID, runID uuid.UUID) (*models.WelfareCheckRun, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			r.id::text,
			r.request_id::text,
			r.patient_id::text,
			r.scheduled_at,
			r.status,
			r.attempts,
			r.last_attempt_at,
			r.next_attempt_at,
			COALESCE(r.livekit_room_name, ''),
			COALESCE(r.livekit_room_sid, ''),
			COALESCE(r.livekit_dispatch_id, ''),
			COALESCE(r.livekit_sip_call_id, ''),
			COALESCE(r.failure_reason, ''),
			r.created_at,
			r.updated_at,
			w.reason_code,
			w.reason_detail,
			w.timezone,
			p.phone_number,
			COALESCE(p.full_name, ''),
			p.user_id::text
		FROM welfare_check_runs r
		INNER JOIN welfare_check_requests w ON w.id = r.request_id
		INNER JOIN patients p ON p.id = r.patient_id
		WHERE r.id = $1
		  AND r.patient_id = $2
	`, runID, patientID)
	run, err := scanWelfareCheckRun(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrWelfareCheckRunNotFound
		}
		return nil, err
	}
	return run, nil
}

func (r *PatientRepository) UpdateWelfareRunLifecycle(ctx context.Context, patientID, runID uuid.UUID, status models.WelfareCheckRunStatus, reason string) (*models.WelfareCheckRun, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var current string
	err = tx.QueryRow(ctx, `
		SELECT status
		FROM welfare_check_runs
		WHERE id = $1 AND patient_id = $2
		FOR UPDATE
	`, runID, patientID).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrWelfareCheckRunNotFound
		}
		return nil, err
	}
	if !isAllowedWelfareRunTransition(models.WelfareCheckRunStatus(current), status) {
		return nil, domainErrors.ErrWelfareCheckRunTransition
	}

	_, err = tx.Exec(ctx, `
		UPDATE welfare_check_runs
		SET status = $3,
		    failure_reason = CASE WHEN $4 = '' THEN failure_reason ELSE $4 END,
		    updated_at = now()
		WHERE id = $1
		  AND patient_id = $2
	`, runID, patientID, string(status), reason)
	if err != nil {
		return nil, err
	}

	requestStatus := ""
	switch status {
	case models.WelfareRunStatusCompleted:
		requestStatus = string(models.WelfareCheckStatusCompleted)
	case models.WelfareRunStatusMissed:
		requestStatus = string(models.WelfareCheckStatusMissed)
	case models.WelfareRunStatusFailed:
		requestStatus = string(models.WelfareCheckStatusFailed)
	}
	if requestStatus != "" {
		_, err = tx.Exec(ctx, `
			UPDATE welfare_check_requests w
			SET status = $3, updated_at = now()
			FROM welfare_check_runs r
			WHERE r.id = $1
			  AND r.patient_id = $2
			  AND w.id = r.request_id
			  AND w.status = 'scheduled'
		`, runID, patientID, requestStatus)
		if err != nil {
			return nil, err
		}
	}

	row := tx.QueryRow(ctx, `
		SELECT
			r.id::text,
			r.request_id::text,
			r.patient_id::text,
			r.scheduled_at,
			r.status,
			r.attempts,
			r.last_attempt_at,
			r.next_attempt_at,
			COALESCE(r.livekit_room_name, ''),
			COALESCE(r.livekit_room_sid, ''),
			COALESCE(r.livekit_dispatch_id, ''),
			COALESCE(r.livekit_sip_call_id, ''),
			COALESCE(r.failure_reason, ''),
			r.created_at,
			r.updated_at,
			w.reason_code,
			w.reason_detail,
			w.timezone,
			p.phone_number,
			COALESCE(p.full_name, ''),
			p.user_id::text
		FROM welfare_check_runs r
		INNER JOIN welfare_check_requests w ON w.id = r.request_id
		INNER JOIN patients p ON p.id = r.patient_id
		WHERE r.id = $1
		  AND r.patient_id = $2
	`, runID, patientID)
	run, err := scanWelfareCheckRun(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return run, nil
}

func isAllowedWelfareRunTransition(from, to models.WelfareCheckRunStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case models.WelfareRunStatusDispatched:
		return to == models.WelfareRunStatusAnswered ||
			to == models.WelfareRunStatusMissed ||
			to == models.WelfareRunStatusFailed ||
			to == models.WelfareRunStatusCompleted
	case models.WelfareRunStatusAnswered:
		return to == models.WelfareRunStatusCompleted ||
			to == models.WelfareRunStatusMissed ||
			to == models.WelfareRunStatusFailed
	case models.WelfareRunStatusClaimed:
		return to == models.WelfareRunStatusFailed || to == models.WelfareRunStatusMissed
	default:
		return false
	}
}

func stableWelfareRoomName(requestID uuid.UUID) string {
	return fmt.Sprintf("welfare-check-%s", requestID.String())
}

type welfareCheckScanner interface {
	Scan(dest ...any) error
}

func scanWelfareCheck(row welfareCheckScanner) (*models.WelfareCheck, error) {
	var (
		id, patientID, reasonCode, status, recurrenceRule string
		check                                             models.WelfareCheck
		recurrenceStartsAt, recurrenceEndsAt, cancelledAt pgtype.Timestamptz
	)
	if err := row.Scan(
		&id,
		&patientID,
		&check.ScheduledAt,
		&check.Timezone,
		&reasonCode,
		&check.ReasonDetail,
		&status,
		&recurrenceRule,
		&recurrenceStartsAt,
		&recurrenceEndsAt,
		&check.CreatedAt,
		&check.UpdatedAt,
		&cancelledAt,
	); err != nil {
		return nil, err
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	parsedPatientID, err := uuid.Parse(patientID)
	if err != nil {
		return nil, err
	}
	check.ID = parsedID
	check.PatientID = parsedPatientID
	check.ReasonCode = models.WelfareCheckReason(reasonCode)
	check.Status = models.WelfareCheckStatus(status)
	check.RecurrenceRule = recurrenceRule
	check.RecurrenceStartsAt = nullableTime(recurrenceStartsAt)
	check.RecurrenceEndsAt = nullableTime(recurrenceEndsAt)
	check.CancelledAt = nullableTime(cancelledAt)
	return &check, nil
}

func scanWelfareCheckWithRun(row welfareCheckScanner) (*models.WelfareCheck, error) {
	var (
		id, patientID, reasonCode, status, recurrenceRule string
		check                                             models.WelfareCheck
		recurrenceStartsAt, recurrenceEndsAt, cancelledAt pgtype.Timestamptz
		latestRunAttempts                                 int32
	)
	if err := row.Scan(
		&id,
		&patientID,
		&check.ScheduledAt,
		&check.Timezone,
		&reasonCode,
		&check.ReasonDetail,
		&status,
		&recurrenceRule,
		&recurrenceStartsAt,
		&recurrenceEndsAt,
		&check.CreatedAt,
		&check.UpdatedAt,
		&cancelledAt,
		&check.LatestRunID,
		&check.LatestRunStatus,
		&latestRunAttempts,
		&check.LatestRunFailureReason,
	); err != nil {
		return nil, err
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	parsedPatientID, err := uuid.Parse(patientID)
	if err != nil {
		return nil, err
	}
	check.ID = parsedID
	check.PatientID = parsedPatientID
	check.ReasonCode = models.WelfareCheckReason(reasonCode)
	check.Status = models.WelfareCheckStatus(status)
	check.RecurrenceRule = recurrenceRule
	check.RecurrenceStartsAt = nullableTime(recurrenceStartsAt)
	check.RecurrenceEndsAt = nullableTime(recurrenceEndsAt)
	check.CancelledAt = nullableTime(cancelledAt)
	check.LatestRunAttempts = latestRunAttempts
	return &check, nil
}

func scanWelfareCheckRun(row welfareCheckScanner) (*models.WelfareCheckRun, error) {
	var (
		id, requestID, patientID, status, userID string
		reasonCode                               string
		run                                      models.WelfareCheckRun
		lastAttemptAt, nextAttemptAt             pgtype.Timestamptz
	)
	if err := row.Scan(
		&id,
		&requestID,
		&patientID,
		&run.ScheduledAt,
		&status,
		&run.Attempts,
		&lastAttemptAt,
		&nextAttemptAt,
		&run.LiveKitRoomName,
		&run.LiveKitRoomSID,
		&run.LiveKitDispatchID,
		&run.LiveKitSIPCallID,
		&run.FailureReason,
		&run.CreatedAt,
		&run.UpdatedAt,
		&reasonCode,
		&run.RequestReasonDetail,
		&run.RequestTimezone,
		&run.PatientPhoneNumber,
		&run.PatientFullName,
		&userID,
	); err != nil {
		return nil, err
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	parsedRequestID, err := uuid.Parse(requestID)
	if err != nil {
		return nil, err
	}
	parsedPatientID, err := uuid.Parse(patientID)
	if err != nil {
		return nil, err
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	run.ID = parsedID
	run.RequestID = parsedRequestID
	run.PatientID = parsedPatientID
	run.PatientUserID = parsedUserID
	run.Status = models.WelfareCheckRunStatus(status)
	run.RequestReasonCode = models.WelfareCheckReason(reasonCode)
	run.LastAttemptAt = nullableTime(lastAttemptAt)
	run.NextAttemptAt = nullableTime(nextAttemptAt)
	return &run, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}
