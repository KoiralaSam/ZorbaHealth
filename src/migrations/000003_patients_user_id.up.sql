-- Add user_id to patients (references users.id in same DB)
ALTER TABLE "patients" ADD COLUMN "user_id" uuid UNIQUE;
CREATE INDEX "patients_user_id_idx" ON "patients" ("user_id");
ALTER TABLE "patients" ADD CONSTRAINT "fk_patients_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id");
