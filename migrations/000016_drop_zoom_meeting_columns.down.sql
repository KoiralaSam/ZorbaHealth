ALTER TABLE scheduled_meetings
ADD COLUMN IF NOT EXISTS zoom_meeting_id text,
ADD COLUMN IF NOT EXISTS host_start_url text;
