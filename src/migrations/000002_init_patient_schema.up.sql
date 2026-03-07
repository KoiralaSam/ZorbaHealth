CREATE TABLE "patients" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "phone_number" varchar(25) NOT NULL,
  "email" varchar(255),
  "full_name" varchar(255),
  "date_of_birth" date,
  "medical_notes" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz
);

CREATE TABLE "calls" (
  "id" bigserial PRIMARY KEY,
  "patient_id" uuid NOT NULL,
  "livekit_room_id" varchar(100) UNIQUE,
  "status" varchar(20),
  "started_at" timestamp,
  "ended_at" timestamp DEFAULT (now()),
  "recording_s3_url" text,
  "summary" text
);

CREATE TABLE "action_items" (
  "id" bigserial PRIMARY KEY,
  "call_id" bigserial NOT NULL,
  "task_description" text,
  "is_completed" boolean DEFAULT false,
  "due_date" timestamp,
  "created_at" timestamp DEFAULT (now())
);

CREATE INDEX ON "patients" ("phone_number");
CREATE INDEX ON "patients" ("medical_notes");
CREATE INDEX ON "calls" ("patient_id");
CREATE INDEX ON "calls" ("livekit_room_id");
CREATE INDEX ON "calls" ("summary");
CREATE INDEX ON "action_items" ("call_id");
CREATE INDEX ON "action_items" ("due_date");
CREATE INDEX ON "action_items" ("call_id", "due_date");

COMMENT ON COLUMN "patients"."phone_number" IS '-- Matched against SIP Caller ID';
COMMENT ON COLUMN "patients"."medical_notes" IS '-- Brief summary for the AI prompt';
COMMENT ON COLUMN "calls"."livekit_room_id" IS '-- maps to livekit SIP room id';
COMMENT ON COLUMN "action_items"."task_description" IS 'Description of the action item from the related call.';

ALTER TABLE "calls" ADD FOREIGN KEY ("patient_id") REFERENCES "patients" ("id");
ALTER TABLE "action_items" ADD FOREIGN KEY ("call_id") REFERENCES "calls" ("id");
