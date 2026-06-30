-- Rollback plan 18.7 custom fields.

DROP INDEX IF EXISTS course.course_enrollments_custom_fields_gin;
DROP INDEX IF EXISTS course.courses_custom_fields_gin;
DROP INDEX IF EXISTS "user".users_custom_fields_gin;

ALTER TABLE course.course_enrollments DROP COLUMN IF EXISTS custom_fields;
ALTER TABLE course.courses DROP COLUMN IF EXISTS custom_fields;
ALTER TABLE "user".users DROP COLUMN IF EXISTS custom_fields;

DROP INDEX IF EXISTS tenant.idx_custom_field_definitions_org_entity;
DROP INDEX IF EXISTS tenant.cfd_org_entity_key;
DROP TABLE IF EXISTS tenant.custom_field_definitions;

DROP TYPE IF EXISTS tenant.custom_field_visibility;
DROP TYPE IF EXISTS tenant.custom_field_type;
DROP TYPE IF EXISTS tenant.custom_field_entity;

ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS custom_fields_enabled;
