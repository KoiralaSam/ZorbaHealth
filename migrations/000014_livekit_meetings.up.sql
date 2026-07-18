ALTER TABLE scheduled_meetings
ADD COLUMN IF NOT EXISTS livekit_room_name text,
ADD COLUMN IF NOT EXISTS livekit_room_sid text,
ADD COLUMN IF NOT EXISTS patient_token text,
ADD COLUMN IF NOT EXISTS staff_token text;

CREATE INDEX IF NOT EXISTS scheduled_meetings_livekit_room_idx ON scheduled_meetings (livekit_room_name);
