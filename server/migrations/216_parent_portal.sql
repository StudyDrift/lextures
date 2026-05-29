-- Plan 13.1: Parent portal — notification preferences table and feature flag.

CREATE TABLE IF NOT EXISTS "user".parent_notification_prefs (
    parent_id            UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    grade_posted         BOOLEAN NOT NULL DEFAULT true,
    missing_assignment   BOOLEAN NOT NULL DEFAULT true,
    low_grade_threshold  INTEGER DEFAULT 70
        CHECK (low_grade_threshold IS NULL OR (low_grade_threshold >= 0 AND low_grade_threshold <= 100)),
    attendance_event     BOOLEAN NOT NULL DEFAULT false,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE "user".parent_notification_prefs IS
    'Plan 13.1: Per-parent email notification subscription preferences.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_parent_portal BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_parent_portal IS
    'Plan 13.1: Enables the parent portal (parent/child linking, read-only grade access, notification prefs).';
