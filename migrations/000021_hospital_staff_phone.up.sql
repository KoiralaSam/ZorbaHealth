-- Add phone_number on hospital_staff for SIP dial-out during bridged calls.
-- Digits-only E.164 without '+', matching patients/users canonical storage (000020).

ALTER TABLE hospital_staff
  ADD COLUMN IF NOT EXISTS phone_number varchar(25);

-- Backfill from linked users.phone_number when present.
UPDATE hospital_staff hs
SET phone_number = regexp_replace(COALESCE(u.phone_number, ''), '[^0-9]', '', 'g')
FROM users u
WHERE hs.user_id = u.id
  AND (hs.phone_number IS NULL OR hs.phone_number = '')
  AND COALESCE(u.phone_number, '') <> '';

ALTER TABLE hospital_staff
  DROP CONSTRAINT IF EXISTS hospital_staff_phone_number_digits_chk;

ALTER TABLE hospital_staff
  ADD CONSTRAINT hospital_staff_phone_number_digits_chk
  CHECK (phone_number IS NULL OR phone_number ~ '^[0-9]{10,15}$');

COMMENT ON COLUMN hospital_staff.phone_number IS
  'Staff PSTN number for LiveKit SIP dial-out into bridged interpretation calls';
