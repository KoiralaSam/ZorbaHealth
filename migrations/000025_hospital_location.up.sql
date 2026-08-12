ALTER TABLE hospitals
  ADD COLUMN IF NOT EXISTS address text,
  ADD COLUMN IF NOT EXISTS city text,
  ADD COLUMN IF NOT EXISTS state text,
  ADD COLUMN IF NOT EXISTS postal_code text,
  ADD COLUMN IF NOT EXISTS phone text;

COMMENT ON COLUMN hospitals.address IS 'Street address used for appointment maps links and directions.';
COMMENT ON COLUMN hospitals.phone IS 'Main hospital phone for patient notifications.';
