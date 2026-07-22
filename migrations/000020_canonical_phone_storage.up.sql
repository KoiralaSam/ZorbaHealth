-- Canonicalize NANP phone numbers to digits-only E.164 without '+':
-- 10-digit national numbers become 11 digits with leading country code 1.
-- Also enforce digits-only storage going forward.

-- Only upgrade 10-digit rows when the 11-digit form is not already present
-- (avoids unique constraint collisions during backfill).
UPDATE patients p
SET phone_number = '1' || phone_number
WHERE phone_number ~ '^[2-9][0-9]{2}[2-9][0-9]{6}$'
  AND NOT EXISTS (
    SELECT 1 FROM patients o
    WHERE o.id <> p.id
      AND regexp_replace(o.phone_number, '[^0-9]', '', 'g') = '1' || p.phone_number
  );

UPDATE users u
SET phone_number = '1' || phone_number
WHERE phone_number ~ '^[2-9][0-9]{2}[2-9][0-9]{6}$'
  AND NOT EXISTS (
    SELECT 1 FROM users o
    WHERE o.id <> u.id
      AND regexp_replace(COALESCE(o.phone_number, ''), '[^0-9]', '', 'g') = '1' || u.phone_number
  );

-- Strip any remaining non-digits that may exist from older writes.
UPDATE patients
SET phone_number = regexp_replace(phone_number, '[^0-9]', '', 'g')
WHERE phone_number ~ '[^0-9]';

UPDATE users
SET phone_number = regexp_replace(phone_number, '[^0-9]', '', 'g')
WHERE phone_number IS NOT NULL AND phone_number ~ '[^0-9]';

ALTER TABLE patients
  DROP CONSTRAINT IF EXISTS patients_phone_number_digits_chk;
ALTER TABLE patients
  ADD CONSTRAINT patients_phone_number_digits_chk
  CHECK (phone_number ~ '^[0-9]{10,15}$');

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_phone_number_digits_chk;
ALTER TABLE users
  ADD CONSTRAINT users_phone_number_digits_chk
  CHECK (phone_number IS NULL OR phone_number ~ '^[0-9]{10,15}$');

COMMENT ON COLUMN patients.phone_number IS
  'Digits-only E.164 without +. NANP stored as 11 digits with country code 1; matched against SIP Caller ID after canonicalization';
COMMENT ON COLUMN users.phone_number IS
  'Digits-only E.164 without +. NANP stored as 11 digits with country code 1';
