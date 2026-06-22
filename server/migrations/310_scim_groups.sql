-- Plan 4.5 — SCIM 2.0 Group provisioning (CRUD, membership, role mapping).

CREATE TABLE IF NOT EXISTS provisioning.scim_groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL,
    external_id     TEXT,
    display_name    TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_groups_institution_display_name
    ON provisioning.scim_groups (institution_id, lower(display_name));

CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_groups_institution_external_id
    ON provisioning.scim_groups (institution_id, external_id)
    WHERE external_id IS NOT NULL AND trim(external_id) <> '';

CREATE INDEX IF NOT EXISTS idx_scim_groups_institution
    ON provisioning.scim_groups (institution_id);

COMMENT ON TABLE provisioning.scim_groups IS 'SCIM 2.0 Group resources provisioned per institution (plan 4.5).';

CREATE TABLE IF NOT EXISTS provisioning.scim_group_members (
    group_id    UUID NOT NULL REFERENCES provisioning.scim_groups (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_scim_group_members_user
    ON provisioning.scim_group_members (user_id);

COMMENT ON TABLE provisioning.scim_group_members IS 'SCIM group membership rows (plan 4.5).';

-- Maps IdP group display names to Lextures entitlements (app role, org role, course enrollment).
CREATE TABLE IF NOT EXISTS provisioning.scim_group_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL,
    display_name    TEXT NOT NULL,
    mapping_kind    TEXT NOT NULL CHECK (mapping_kind IN ('app_role', 'org_role', 'course_enrollment')),
    app_role_name   TEXT,
    org_role_key    TEXT,
    course_code     TEXT,
    enrollment_role TEXT NOT NULL DEFAULT 'student',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scim_group_mappings_org_role_fkey
        FOREIGN KEY (org_role_key) REFERENCES "user".org_role_keys (role_key),
    CONSTRAINT scim_group_mappings_enrollment_role_fkey
        FOREIGN KEY (enrollment_role) REFERENCES course.enrollment_roles (role_key),
    CONSTRAINT scim_group_mappings_kind_fields CHECK (
        (mapping_kind = 'app_role' AND app_role_name IS NOT NULL AND org_role_key IS NULL AND course_code IS NULL)
        OR (mapping_kind = 'org_role' AND org_role_key IS NOT NULL AND app_role_name IS NULL AND course_code IS NULL)
        OR (mapping_kind = 'course_enrollment' AND course_code IS NOT NULL AND app_role_name IS NULL AND org_role_key IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_group_mappings_app_role
    ON provisioning.scim_group_mappings (institution_id, lower(display_name), mapping_kind, lower(app_role_name))
    WHERE mapping_kind = 'app_role';

CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_group_mappings_org_role
    ON provisioning.scim_group_mappings (institution_id, lower(display_name), mapping_kind, org_role_key)
    WHERE mapping_kind = 'org_role';

CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_group_mappings_course
    ON provisioning.scim_group_mappings (institution_id, lower(display_name), mapping_kind, lower(course_code), enrollment_role)
    WHERE mapping_kind = 'course_enrollment';

COMMENT ON TABLE provisioning.scim_group_mappings IS 'SCIM group display name → Lextures role/org-role/enrollment mapping (plan 4.5).';

-- Extend audit log operations for group membership changes.
ALTER TABLE provisioning.scim_provisioning_events
    DROP CONSTRAINT IF EXISTS scim_provisioning_events_operation_check;

ALTER TABLE provisioning.scim_provisioning_events
    ADD CONSTRAINT scim_provisioning_events_operation_check
    CHECK (operation IN ('create', 'update', 'deactivate', 'delete', 'member_add', 'member_remove'));