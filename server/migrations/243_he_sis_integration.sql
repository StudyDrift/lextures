-- Plan 14.1: Higher-ed SIS vendors (Banner, Workday, Colleague, Jenzabar, PeopleSoft).

ALTER TABLE sis.sis_connections
    DROP CONSTRAINT IF EXISTS sis_connections_vendor_check;

ALTER TABLE sis.sis_connections
    ADD CONSTRAINT sis_connections_vendor_check
    CHECK (vendor IN (
        'powerschool', 'infinite_campus', 'skyward', 'aeries',
        'banner', 'workday', 'colleague', 'jenzabar', 'peoplesoft'
    ));

COMMENT ON TABLE sis.sis_connections IS
    'SIS vendor connection configs per org (K-12 plan 13.7 + HE plan 14.1). Credentials stored via secrets manager refs.';

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS sis_section_id TEXT;
ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS sis_synced_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS enrollments_sis_section_idx
    ON course.course_enrollments (sis_section_id) WHERE sis_section_id IS NOT NULL;

COMMENT ON COLUMN course.course_enrollments.sis_section_id IS
    'Plan 14.1: SIS section identifier for HE roster sync.';
COMMENT ON COLUMN course.course_enrollments.sis_synced_at IS
    'Plan 14.1: Timestamp of last successful SIS sync for this enrollment.';
