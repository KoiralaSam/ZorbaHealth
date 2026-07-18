ALTER TABLE scheduled_meetings
DROP COLUMN IF EXISTS livekit_room_name,
DROP COLUMN IF EXISTS livekit_room_sid,
DROP COLUMN IF EXISTS patient_token,
DROP COLUMN IF EXISTS staff_token;
