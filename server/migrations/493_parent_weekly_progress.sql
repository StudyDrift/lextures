-- Issue #776 — one-click unsubscribe can mark a preference as user-chosen,
-- and weekly parent progress sends are recorded once per local week.

ALTER TABLE settings.notification_preferences
  ADD COLUMN IF NOT EXISTS user_overridden BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN settings.notification_preferences.user_overridden IS
  'True after the user saves or unsubscribes this event. Homeschool weekly progress stays on until then.';

CREATE TABLE IF NOT EXISTS settings.parent_weekly_progress_sends (
  parent_user_id UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  week_start     DATE NOT NULL,
  sent_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (parent_user_id, week_start)
);

COMMENT ON TABLE settings.parent_weekly_progress_sends IS
  'Issue #776: one weekly parent progress email per parent per local send date.';
