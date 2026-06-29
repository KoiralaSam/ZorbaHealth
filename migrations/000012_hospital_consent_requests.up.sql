CREATE TABLE "hospital_consent_requests" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "token" text UNIQUE NOT NULL,
  "hospital_id" uuid NOT NULL,
  "staff_id" uuid NOT NULL,
  "patient_id" uuid,
  "requested_permissions" text NOT NULL,
  "note" text,
  "expires_at" timestamptz NOT NULL,
  "approved_at" timestamptz,
  "approved_patient_id" uuid,
  "created_at" timestamptz DEFAULT (now())
);

CREATE INDEX "hospital_consent_requests_hospital_idx" ON "hospital_consent_requests" ("hospital_id", "created_at" DESC);
CREATE INDEX "hospital_consent_requests_patient_idx" ON "hospital_consent_requests" ("approved_patient_id", "created_at" DESC);
CREATE INDEX "hospital_consent_requests_token_idx" ON "hospital_consent_requests" ("token");

ALTER TABLE "hospital_consent_requests" ADD FOREIGN KEY ("hospital_id") REFERENCES "hospitals" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "hospital_consent_requests" ADD FOREIGN KEY ("staff_id") REFERENCES "hospital_staff" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "hospital_consent_requests" ADD FOREIGN KEY ("patient_id") REFERENCES "patients" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "hospital_consent_requests" ADD FOREIGN KEY ("approved_patient_id") REFERENCES "patients" ("id") DEFERRABLE INITIALLY IMMEDIATE;

COMMENT ON TABLE "hospital_consent_requests" IS 'Staff-created QR consent requests. Patient approval creates or reactivates patient_hospital_consents.';
