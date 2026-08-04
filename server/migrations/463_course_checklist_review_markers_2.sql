-- CC.5: Review markers for accommodations and integrity settings.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS accommodations_reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS integrity_settings_reviewed_at TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.accommodations_reviewed_at IS
    'Set when course staff open/save the course accommodations surface; drives accommodations.reviewed.';
COMMENT ON COLUMN course.courses.integrity_settings_reviewed_at IS
    'Set when course staff save integrity settings; drives integrity.high-stakes-settings.';
