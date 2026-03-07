ALTER TABLE "patients" DROP CONSTRAINT IF EXISTS "fk_patients_user_id";
DROP INDEX IF EXISTS "patients_user_id_idx";
ALTER TABLE "patients" DROP COLUMN IF EXISTS "user_id";
