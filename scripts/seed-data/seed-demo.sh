#!/usr/bin/env bash
# Seeds a demo patient and demo hospital staff account into the local dev
# cluster so authenticated flows (and scripts/evaluation/demo-smoke.mjs) work
# without going through email/OTP registration.
#
# Idempotent: safe to re-run; it updates password hashes in place.
#
# Requirements: kubectl context pointing at the local cluster (namespace dev),
# Go toolchain (for bcrypt hashing), postgres running in-cluster.
#
# Defaults match scripts/evaluation/demo-smoke.mjs. Override via env:
#   DEMO_PATIENT_PHONE, DEMO_PATIENT_EMAIL, DEMO_PATIENT_PASSWORD,
#   DEMO_HOSPITAL_EMAIL, DEMO_HOSPITAL_PASSWORD

set -euo pipefail

cd "$(dirname "$0")/../.."

NAMESPACE="dev"
PATIENT_PHONE="${DEMO_PATIENT_PHONE:-+15555550100}"
PATIENT_EMAIL="${DEMO_PATIENT_EMAIL:-demo.patient@zorbahealth.local}"
PATIENT_PASSWORD="${DEMO_PATIENT_PASSWORD:-demo-password}"
PATIENT_NAME="Demo Patient"
STAFF_EMAIL="${DEMO_HOSPITAL_EMAIL:-staff@zorbahealth.local}"
STAFF_PASSWORD="${DEMO_HOSPITAL_PASSWORD:-demo-password}"
STAFF_NAME="Demo Staff"
HOSPITAL_NAME="Zorba Demo Hospital"
HOSPITAL_LICENSE="DEMO-LICENSE-0001"

# The DB stores canonical digits-only phone numbers (see migration 000020);
# login input like "+1 (555) 555-0100" is normalized to the same form.
PATIENT_PHONE_CANONICAL="$(printf '%s' "$PATIENT_PHONE" | tr -cd '0-9')"

echo "Hashing demo passwords (bcrypt cost 10)..."
PATIENT_HASH="$(go run ./scripts/seed-data/hashpw "$PATIENT_PASSWORD")"
STAFF_HASH="$(go run ./scripts/seed-data/hashpw "$STAFF_PASSWORD")"

PGUSER="$(kubectl -n "$NAMESPACE" get secret postgres-secret -o jsonpath='{.data.POSTGRES_USER}' | base64 -d)"
PGDB="$(kubectl -n "$NAMESPACE" get secret postgres-secret -o jsonpath='{.data.POSTGRES_DB}' | base64 -d)"

echo "Seeding demo accounts into $PGDB as $PGUSER..."
kubectl -n "$NAMESPACE" exec -i deploy/postgres -- \
  psql -v ON_ERROR_STOP=1 -U "$PGUSER" -d "$PGDB" <<EOF
DO \$seed\$
DECLARE
  v_patient_user uuid;
  v_patient uuid;
  v_hospital uuid;
  v_staff_user uuid;
BEGIN
  -- Demo patient auth user
  SELECT id INTO v_patient_user FROM users
  WHERE phone_number = '${PATIENT_PHONE_CANONICAL}' OR email = '${PATIENT_EMAIL}'
  LIMIT 1;
  IF v_patient_user IS NULL THEN
    INSERT INTO users (email, phone_number, password_hash, role)
    VALUES ('${PATIENT_EMAIL}', '${PATIENT_PHONE_CANONICAL}', '${PATIENT_HASH}', 'patient')
    RETURNING id INTO v_patient_user;
  ELSE
    UPDATE users
    SET password_hash = '${PATIENT_HASH}', role = 'patient'
    WHERE id = v_patient_user;
  END IF;

  -- Demo patient profile
  SELECT id INTO v_patient FROM patients WHERE user_id = v_patient_user;
  IF v_patient IS NULL THEN
    INSERT INTO patients (user_id, phone_number, email, full_name)
    VALUES (v_patient_user, '${PATIENT_PHONE_CANONICAL}', '${PATIENT_EMAIL}', '${PATIENT_NAME}')
    RETURNING id INTO v_patient;
  END IF;

  -- Demo hospital
  SELECT id INTO v_hospital FROM hospitals WHERE license_no = '${HOSPITAL_LICENSE}';
  IF v_hospital IS NULL THEN
    INSERT INTO hospitals (name, license_no, active)
    VALUES ('${HOSPITAL_NAME}', '${HOSPITAL_LICENSE}', true)
    RETURNING id INTO v_hospital;
  END IF;

  -- Demo staff auth user (auth happens against users; hospital_staff
  -- password_hash stays empty, matching the registration path).
  SELECT id INTO v_staff_user FROM users WHERE email = '${STAFF_EMAIL}' LIMIT 1;
  IF v_staff_user IS NULL THEN
    INSERT INTO users (email, password_hash, role)
    VALUES ('${STAFF_EMAIL}', '${STAFF_HASH}', 'hospital_staff')
    RETURNING id INTO v_staff_user;
  ELSE
    UPDATE users
    SET password_hash = '${STAFF_HASH}', role = 'hospital_staff'
    WHERE id = v_staff_user;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM hospital_staff WHERE email = '${STAFF_EMAIL}') THEN
    INSERT INTO hospital_staff (hospital_id, user_id, email, password_hash, name, role, active)
    VALUES (v_hospital, v_staff_user, '${STAFF_EMAIL}', '', '${STAFF_NAME}', 'doctor', true);
  ELSE
    UPDATE hospital_staff
    SET hospital_id = v_hospital, user_id = v_staff_user, active = true
    WHERE email = '${STAFF_EMAIL}';
  END IF;

  -- Link patient to hospital so portal and console flows work
  IF NOT EXISTS (
    SELECT 1 FROM patient_hospital_consents
    WHERE patient_id = v_patient AND hospital_id = v_hospital AND revoked_at IS NULL
  ) THEN
    INSERT INTO patient_hospital_consents (patient_id, hospital_id)
    VALUES (v_patient, v_hospital);
  END IF;

  RAISE NOTICE 'demo patient id: %', v_patient;
  RAISE NOTICE 'demo hospital id: %', v_hospital;
END
\$seed\$;
EOF

echo
echo "Demo credentials seeded:"
echo "  Patient login (phone):  $PATIENT_PHONE / $PATIENT_PASSWORD"
echo "  Patient login (email):  $PATIENT_EMAIL / $PATIENT_PASSWORD"
echo "  Hospital staff login:   $STAFF_EMAIL / $STAFF_PASSWORD"
