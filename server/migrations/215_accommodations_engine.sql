-- Plan 12.10: Accommodations engine (IEP/504 compliance, audit log, display prefs).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS accommodations_engine_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_accommodations_engine BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.accommodations_engine_enabled IS
    'Plan 12.10: Enables the K-12 accommodations engine (profiles, quiz enforcement, audit log).';
COMMENT ON COLUMN settings.platform_app_settings.ff_accommodations_engine IS
    'Plan 12.10: When true, writes accommodation application events to accommodation_audit_log.';

INSERT INTO "user".app_roles (name, description, scope)
VALUES (
        'SPED Coordinator',
        'Manages student IEP/504 accommodation records for K-12 compliance.',
        'global'
    )
ON CONFLICT (name) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id,
       p.id
FROM "user".app_roles r
JOIN "user".permissions p ON p.permission_string = 'global:user:accommodations:manage'
WHERE r.name = 'SPED Coordinator'
ON CONFLICT (role_id, permission_id) DO NOTHING;

ALTER TABLE course.student_accommodations
    ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS dyslexia_display_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS high_contrast_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS reduced_motion_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS separate_setting BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN course.student_accommodations.tts_enabled IS
    'Plan 12.10: Auto-enables text-to-speech read-aloud for this learner.';
COMMENT ON COLUMN course.student_accommodations.dyslexia_display_enabled IS
    'Plan 12.10: Applies dyslexia-friendly font and spacing preset.';
COMMENT ON COLUMN course.student_accommodations.high_contrast_enabled IS
    'Plan 12.10: Applies high-contrast theme override.';
COMMENT ON COLUMN course.student_accommodations.reduced_motion_enabled IS
    'Plan 12.10: Applies reduced-motion display override.';
COMMENT ON COLUMN course.student_accommodations.separate_setting IS
    'Plan 12.10: Informational flag for separate testing environment (no v1 enforcement).';
COMMENT ON COLUMN course.student_accommodations.active IS
    'Plan 12.10: Soft-delete flag; inactive rows are ignored for enforcement.';

DROP INDEX IF EXISTS course.uq_student_accommodations_user_global;
DROP INDEX IF EXISTS course.uq_student_accommodations_user_course;

CREATE UNIQUE INDEX uq_student_accommodations_user_global ON course.student_accommodations (user_id)
WHERE
    course_id IS NULL
    AND active = true;

CREATE UNIQUE INDEX uq_student_accommodations_user_course ON course.student_accommodations (user_id, course_id)
WHERE
    course_id IS NOT NULL
    AND active = true;

ALTER TABLE course.quiz_attempts
    ADD COLUMN IF NOT EXISTS effective_time_limit_seconds INTEGER;

COMMENT ON COLUMN course.quiz_attempts.effective_time_limit_seconds IS
    'Plan 12.10: Server-enforced time limit in seconds after accommodation multiplier (null = untimed).';

CREATE TABLE IF NOT EXISTS course.accommodation_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    student_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    accommodation_type TEXT NOT NULL,
    value_applied JSONB NOT NULL DEFAULT '{}',
    context TEXT NOT NULL,
    context_id UUID,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accommodation_audit_log_student ON course.accommodation_audit_log (student_id, applied_at DESC);

COMMENT ON TABLE course.accommodation_audit_log IS
    'Append-only IDEA compliance log of accommodation applications (no UPDATE/DELETE).';

ALTER TABLE settings.user_reading_preferences
    ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS dyslexia_display_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS high_contrast_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS reduced_motion_enabled BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN settings.user_reading_preferences.tts_enabled IS
    'Plan 12.10: User opt-in for text-to-speech read-aloud.';
COMMENT ON COLUMN settings.user_reading_preferences.dyslexia_display_enabled IS
    'Plan 12.10: User opt-in for dyslexia-friendly display preset.';
COMMENT ON COLUMN settings.user_reading_preferences.high_contrast_enabled IS
    'Plan 12.10: User opt-in for high-contrast theme.';
COMMENT ON COLUMN settings.user_reading_preferences.reduced_motion_enabled IS
    'Plan 12.10: User opt-in for reduced-motion display.';
