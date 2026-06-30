-- Plan 18.7: Custom field definitions and JSONB value storage on users, courses, enrollments.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS custom_fields_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.custom_fields_enabled IS
    'Plan 18.7: Enables org-admin custom field schemas and value APIs (/org-admin/custom-fields).';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'custom_field_entity' AND typnamespace = 'tenant'::regnamespace) THEN
        CREATE TYPE tenant.custom_field_entity AS ENUM ('user', 'course', 'enrollment');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'custom_field_type' AND typnamespace = 'tenant'::regnamespace) THEN
        CREATE TYPE tenant.custom_field_type AS ENUM ('text', 'number', 'boolean', 'date', 'select');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'custom_field_visibility' AND typnamespace = 'tenant'::regnamespace) THEN
        CREATE TYPE tenant.custom_field_visibility AS ENUM ('admin_only', 'instructor', 'student');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS tenant.custom_field_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    entity_type     tenant.custom_field_entity NOT NULL,
    key             TEXT NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    label           TEXT NOT NULL,
    field_type      tenant.custom_field_type NOT NULL,
    select_options  TEXT[],
    is_required     BOOLEAN NOT NULL DEFAULT FALSE,
    visibility      tenant.custom_field_visibility NOT NULL DEFAULT 'admin_only',
    sort_order      INT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS cfd_org_entity_key
    ON tenant.custom_field_definitions (org_id, entity_type, key)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cfd_org_entity_sort
    ON tenant.custom_field_definitions (org_id, entity_type, sort_order)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE tenant.custom_field_definitions IS 'Org-scoped custom field schemas (plan 18.7).';

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_users_custom_fields_gin
    ON "user".users USING gin (custom_fields);

CREATE INDEX IF NOT EXISTS idx_courses_custom_fields_gin
    ON course.courses USING gin (custom_fields);

CREATE INDEX IF NOT EXISTS idx_course_enrollments_custom_fields_gin
    ON course.course_enrollments USING gin (custom_fields);
