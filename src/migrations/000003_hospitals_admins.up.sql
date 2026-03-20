CREATE TABLE "hospitals" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "name" text NOT NULL,
  "license_no" text UNIQUE NOT NULL,
  "active" boolean DEFAULT true,
  "created_at" timestamptz DEFAULT (now())
);

CREATE TABLE "hospital_staff" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "hospital_id" uuid NOT NULL,
  "user_id" uuid UNIQUE,
  "email" text UNIQUE NOT NULL,
  "password_hash" text NOT NULL,
  "name" text NOT NULL,
  "role" text NOT NULL,
  "active" boolean DEFAULT true,
  "created_at" timestamptz DEFAULT (now())
);

CREATE TABLE "patient_hospital_consents" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "patient_id" uuid NOT NULL,
  "hospital_id" uuid NOT NULL,
  "granted_at" timestamptz DEFAULT (now()),
  "revoked_at" timestamptz
);

CREATE TABLE "admins" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid UNIQUE NOT NULL,
  "email" varchar(255),
  "name" text NOT NULL,
  "active" boolean DEFAULT true,
  "created_at" timestamptz DEFAULT (now())
);

CREATE INDEX "hospital_staff_user_id_idx" ON "hospital_staff" ("user_id");
CREATE INDEX "consents_patient_idx" ON "patient_hospital_consents" ("patient_id");
CREATE INDEX "consents_hospital_idx" ON "patient_hospital_consents" ("hospital_id");
CREATE UNIQUE INDEX "consents_patient_hospital_unique" ON "patient_hospital_consents" ("patient_id", "hospital_id");
CREATE INDEX "admins_user_id_idx" ON "admins" ("user_id");

COMMENT ON COLUMN "hospital_staff"."user_id" IS 'links staff member to auth system';
COMMENT ON COLUMN "hospital_staff"."role" IS 'doctor | nurse | admin | billing';
COMMENT ON COLUMN "patient_hospital_consents"."revoked_at" IS 'NULL = active consent. Set to revoke — never deleted.';

ALTER TABLE "hospital_staff" ADD FOREIGN KEY ("hospital_id") REFERENCES "hospitals" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "hospital_staff" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "patient_hospital_consents" ADD FOREIGN KEY ("patient_id") REFERENCES "patients" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "patient_hospital_consents" ADD FOREIGN KEY ("hospital_id") REFERENCES "hospitals" ("id") DEFERRABLE INITIALLY IMMEDIATE;
ALTER TABLE "admins" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
