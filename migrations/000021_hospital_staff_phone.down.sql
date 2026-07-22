ALTER TABLE hospital_staff DROP CONSTRAINT IF EXISTS hospital_staff_phone_number_digits_chk;
ALTER TABLE hospital_staff DROP COLUMN IF EXISTS phone_number;
