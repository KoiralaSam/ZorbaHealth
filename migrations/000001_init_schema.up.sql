CREATE TABLE "users" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "email" varchar(255) UNIQUE,
  "phone_number" varchar(25) UNIQUE,
  "password_hash" text,
  "role" varchar(30) NOT NULL,
  "created_at" timestamptz DEFAULT (now())
);

CREATE TABLE "auths" (
  "id" BIGSERIAL PRIMARY KEY,
  "user_id" uuid NOT NULL,
  "auth_uuid" uuid NOT NULL DEFAULT (gen_random_uuid())
);

CREATE INDEX "users_role_idx" ON "users" ("role");
CREATE INDEX "auths_user_id_auth_uuid_idx" ON "auths" ("user_id", "auth_uuid");
COMMENT ON COLUMN "users"."role" IS 'patient | health_service | admin';
ALTER TABLE "auths" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
