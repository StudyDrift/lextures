-- Plan 14.8 — HE plagiarism workflow: feature flag + per-course settings.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_plagiarism_checks BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_plagiarism_checks IS
    'Enables HE plagiarism workflow: async scans, originality API, course settings (plan 14.8).';

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS plagiarism_checks_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS plagiarism_provider TEXT
        CHECK (plagiarism_provider IS NULL OR plagiarism_provider IN ('none', 'turnitin', 'copyleaks', 'gptzero')),
    ADD COLUMN IF NOT EXISTS plagiarism_alert_threshold_pct NUMERIC(5, 2) NOT NULL DEFAULT 40;

COMMENT ON COLUMN course.courses.plagiarism_checks_enabled IS
    'When false, no originality scans run for assignments in this course (plan 14.8).';
COMMENT ON COLUMN course.courses.plagiarism_provider IS
    'Optional course-level override for the active external originality provider.';
COMMENT ON COLUMN course.courses.plagiarism_alert_threshold_pct IS
    'Similarity percentage at which instructors see a highlighted alert (plan 14.8).';
