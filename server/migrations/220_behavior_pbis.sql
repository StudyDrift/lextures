-- Plan 13.3: Behavior / PBIS tracking (positive recognitions, negative referrals).

CREATE SCHEMA IF NOT EXISTS behavior;

CREATE TABLE IF NOT EXISTS behavior.categories (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id    UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    name      TEXT        NOT NULL,
    type      TEXT        NOT NULL CHECK (type IN ('positive', 'negative')),
    color     TEXT,
    active    BOOLEAN     NOT NULL DEFAULT true,
    UNIQUE (org_id, name)
);

CREATE INDEX IF NOT EXISTS idx_behavior_categories_org
    ON behavior.categories (org_id);

COMMENT ON TABLE behavior.categories IS
    'Plan 13.3: Per-org configurable behavior categories (PBIS positive and negative types).';

CREATE TABLE IF NOT EXISTS behavior.pbis_awards (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    awarded_by  UUID        NOT NULL REFERENCES "user".users (id),
    category_id UUID        NOT NULL REFERENCES behavior.categories (id),
    org_id      UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    points      INTEGER     NOT NULL DEFAULT 1 CHECK (points > 0),
    note        TEXT,
    awarded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pbis_awards_student
    ON behavior.pbis_awards (student_id, awarded_at DESC);
CREATE INDEX IF NOT EXISTS idx_pbis_awards_org_date
    ON behavior.pbis_awards (org_id, awarded_at DESC);

COMMENT ON TABLE behavior.pbis_awards IS
    'Plan 13.3: PBIS positive point awards per student. FERPA education record.';

CREATE TABLE IF NOT EXISTS behavior.referrals (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    filed_by    UUID        NOT NULL REFERENCES "user".users (id),
    org_id      UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    school_id   UUID        REFERENCES tenant.org_units (id) ON DELETE SET NULL,
    category_id UUID        NOT NULL REFERENCES behavior.categories (id),
    incident_at TIMESTAMPTZ NOT NULL,
    location    TEXT,
    description TEXT        NOT NULL,
    response    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_behavior_referrals_student
    ON behavior.referrals (student_id, incident_at DESC);
CREATE INDEX IF NOT EXISTS idx_behavior_referrals_org
    ON behavior.referrals (org_id, incident_at DESC);

COMMENT ON TABLE behavior.referrals IS
    'Plan 13.3: Negative behavior referral records. FERPA-protected; description field access is restricted.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_behavior_pbis BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_behavior_pbis IS
    'Plan 13.3: Enables PBIS behavior tracking (positive points and negative referrals).';
