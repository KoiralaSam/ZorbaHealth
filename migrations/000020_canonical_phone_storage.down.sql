ALTER TABLE patients DROP CONSTRAINT IF EXISTS patients_phone_number_digits_chk;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_number_digits_chk;

COMMENT ON COLUMN patients.phone_number IS 'Matched against SIP Caller ID';
COMMENT ON COLUMN users.phone_number IS NULL;

-- Note: data rewritten to 11-digit NANP form is intentionally not reverted.
