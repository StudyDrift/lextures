-- Plan 18.7: Custom fields / metadata on user, course, and enrollment.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS custom_fields_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.custom_fields_enabled IS
    'Plan 18.7: Enables org-admin custom field definitions and values on users, courses, and enrollments.';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'custom_field_entity') THEN
        CREATE TYPE tenant.custom_field_entity AS ENUM ('user', 'course', 'enrollment');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'custom_field_type') THEN
        CREATE TYPE tenant.custom_field_type AS ENUM ('text', 'number', 'boolean', 'date', 'select');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'custom_field_visibility') THEN
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
    is_required     BOOLEAN NOT NULL DEFAULT false,
    visibility      tenant.custom_field_visibility NOT NULL DEFAULT 'admin_only',
    sort_order      INT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cfd_org_entity_key
    ON tenant.custom_field_definitions (org_id, entity_type, key)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_custom_field_definitions_org_entity
    ON tenant.custom_field_definitions (org_id, entity_type, sort_order)
    WHERE deleted_at IS NULL;

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}';

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}';

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS users_custom_fields_gin
    ON "user".users USING GIN (custom_fields);

CREATE INDEX IF NOT EXISTS courses_custom_fields_gin
    ON course.courses USING GIN (custom_fields);

CREATE INDEX IF NOT EXISTS course_enrollments_custom_fields_gin
    ON course.course_enrollments USING GIN (custom_fields);

COMMENT ON TABLE tenant.custom_field_definitions IS
    'Org-scoped custom field schema definitions for users, courses, and enrollments (plan 18.7).';

COMMENT ON COLUMN "user".users.custom_fields IS
    'JSONB map of custom field key to value (plan 18.7).';

COMMENT ON COLUMN course.courses.custom_fields IS
    'JSONB map of custom field key to value (plan 18.7).';

COMMENT ON COLUMN course.course_enrollments.custom_fields IS
    'JSONB map of custom field key to value (plan 18.7).';
