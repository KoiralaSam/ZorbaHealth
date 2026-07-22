DROP INDEX IF EXISTS scheduled_meetings_reminder_due_idx;
ALTER TABLE scheduled_meetings DROP COLUMN IF EXISTS reminder_sent_at;
