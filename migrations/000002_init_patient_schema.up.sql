CREATE TABLE "patients" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid UNIQUE,
  "phone_number" varchar(25) NOT NULL,
  "email" varchar(255),
  "full_name" varchar(255),
  "date_of_birth" date,
  "medical_notes" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz
);

CREATE TABLE "calls" (
  "id" BIGSERIAL PRIMARY KEY,
  "patient_id" uuid NOT NULL,
  "livekit_room_id" varchar(100) UNIQUE,
  "status" varchar(20),
  "started_at" timestamp,
  "ended_at" timestamp DEFAULT (now()),
  "recording_s3_url" text,
  "summary" text
);

CREATE INDEX "patients_phone_number_idx" ON "patients" ("phone_number");
CREATE INDEX "patients_medical_notes_idx" ON "patients" ("medical_notes");
CREATE INDEX "patients_user_id_idx" ON "patients" ("user_id");
CREATE INDEX "calls_patient_id_idx" ON "calls" ("patient_id");
CREATE INDEX "calls_livekit_room_id_idx" ON "calls" ("livekit_room_id");
CREATE INDEX "calls_summary_idx" ON "calls" ("summary");

COMMENT ON COLUMN "patients"."user_id" IS 'linked after registration — added in migration 000003';
COMMENT ON COLUMN "patients"."phone_number" IS 'Matched against SIP Caller ID';
COMMENT ON COLUMN "patients"."medical_notes" IS 'Brief summary for the AI prompt';
COMMENT ON COLUMN "calls"."livekit_room_id" IS 'maps to livekit SIP room id';
COMMENT ON COLUMN "calls"."status" IS 'active | ended | failed';

ALTER TABLE "patients" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "calls" ADD FOREIGN KEY ("patient_id") REFERENCES "patients" ("id") DEFERRABLE INITIALLY IMMEDIATE;
