-- Multi-campus / consortium course sharing (plan 14.18).

CREATE TABLE tenant.consortium_agreements (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_org_id  UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    guest_org_id UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'active', 'terminated')),
    signed_at    TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (host_org_id, guest_org_id),
    CHECK (host_org_id <> guest_org_id)
);

CREATE INDEX idx_consortium_agreements_host ON tenant.consortium_agreements (host_org_id, status);
CREATE INDEX idx_consortium_agreements_guest ON tenant.consortium_agreements (guest_org_id, status);

COMMENT ON TABLE tenant.consortium_agreements IS
    'Formal sharing agreements between host and guest institutions (plan 14.18).';

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS consortium_shareable BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.consortium_shareable IS
    'When true, partner institutions with an active agreement may enroll students (plan 14.18).';

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS home_org_id UUID REFERENCES tenant.organizations (id);

CREATE INDEX idx_course_enrollments_home_org ON course.course_enrollments (home_org_id)
    WHERE home_org_id IS NOT NULL;

COMMENT ON COLUMN course.course_enrollments.home_org_id IS
    'Guest institution for cross-institutional enrollments; NULL for native enrollments (plan 14.18).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_consortium_sharing BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_consortium_sharing IS
    'Enables multi-campus consortium course sharing and cross-institutional enrollment (plan 14.18).';
