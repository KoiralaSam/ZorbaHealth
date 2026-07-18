ALTER TABLE scheduled_meetings
DROP COLUMN IF EXISTS zoom_meeting_id,
DROP COLUMN IF EXISTS host_start_url;
