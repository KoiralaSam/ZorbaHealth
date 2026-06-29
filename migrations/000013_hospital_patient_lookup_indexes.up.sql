CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS patients_email_lower_idx
ON patients (lower(email))
WHERE email IS NOT NULL;

CREATE INDEX IF NOT EXISTS patients_full_name_trgm_idx
ON patients USING gin (full_name gin_trgm_ops)
WHERE full_name IS NOT NULL;

CREATE INDEX IF NOT EXISTS consents_hospital_active_patient_idx
ON patient_hospital_consents (hospital_id, patient_id, granted_at DESC)
WHERE revoked_at IS NULL;
