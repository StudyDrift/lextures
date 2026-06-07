-- Plan 13.13: Free/reduced-lunch & demographic flags (Title I reporting).
-- Depends on: compliance schema (179_ferpa), tenant.org_units (129), settings.platform_app_settings (118).

CREATE TABLE IF NOT EXISTS compliance.student_demographics (
    student_id          UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    free_lunch          BOOLEAN,
    reduced_lunch       BOOLEAN,
    ell_status          BOOLEAN,
    disability_status   BOOLEAN,
    race_ethnicity_code TEXT CHECK (race_ethnicity_code IS NULL OR race_ethnicity_code IN ('1','2','3','4','5','6','7')),
    homeless_indicator  BOOLEAN,
    migrant_indicator   BOOLEAN,
    data_source         TEXT NOT NULL DEFAULT 'manual' CHECK (data_source IN ('sis_sync', 'manual')),
    last_verified_at    TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_student_demographics_updated
    ON compliance.student_demographics (updated_at DESC);

COMMENT ON TABLE compliance.student_demographics IS
    'Plan 13.13: Student demographic flags for Title I reporting. FERPA/NSLP protected; access restricted to authorized school officials.';

-- Org role for data analysts (plan 13.13).
INSERT INTO "user".org_role_keys (role_key, display_name, is_admin, is_viewer, sort_order)
VALUES ('data_analyst', 'Data Analyst', false, true, 30)
ON CONFLICT (role_key) DO NOTHING;

-- RBAC permissions for demographic data access.
INSERT INTO "user".permissions (permission_string, description)
VALUES
    ('compliance:demographics:read:*', 'May view individual student demographic records (plan 13.13).'),
    ('compliance:demographics:write:*', 'May manually update student demographic records (plan 13.13).'),
    ('compliance:demographics:report:*', 'May run aggregate Title I demographic reports (plan 13.13).')
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
JOIN "user".permissions p ON p.permission_string IN (
    'compliance:demographics:read:*',
    'compliance:demographics:write:*',
    'compliance:demographics:report:*'
)
WHERE r.name = 'Global Admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_demographics BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_demographics IS
    'Plan 13.13: Enables student demographic flags and Title I aggregate reporting.';
