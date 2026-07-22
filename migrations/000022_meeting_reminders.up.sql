ALTER TABLE scheduled_meetings
  ADD COLUMN IF NOT EXISTS reminder_sent_at timestamptz;

CREATE INDEX IF NOT EXISTS scheduled_meetings_reminder_due_idx
  ON scheduled_meetings (starts_at)
  WHERE status = 'scheduled' AND reminder_sent_at IS NULL;

COMMENT ON COLUMN scheduled_meetings.reminder_sent_at IS
  'Set when the 15-minute-before meeting reminder email was sent';
