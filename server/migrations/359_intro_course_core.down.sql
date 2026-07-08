-- Revert IC01 intro course foundation (retains course row and system users for manual cleanup).

DROP INDEX IF EXISTS course.idx_courses_short_code;
ALTER TABLE course.courses DROP COLUMN IF EXISTS short_code;

ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS intro_course_enabled;

-- Restore account_type constraint without 'system' (system users remain with account_type='system').
ALTER TABLE "user".users DROP CONSTRAINT IF EXISTS users_account_type_check;
ALTER TABLE "user".users
    ADD CONSTRAINT users_account_type_check
        CHECK (account_type IN ('standard', 'parent'));

CREATE OR REPLACE FUNCTION tenant.count_learner_seats(p_org_id UUID)
RETURNS INT
LANGUAGE sql
STABLE
AS $$
    SELECT COUNT(*)::INT
    FROM "user".users u
    WHERE u.org_id = p_org_id
      AND u.deactivated_at IS NULL
      AND NOT u.login_blocked
      AND NOT EXISTS (
          SELECT 1 FROM "user".org_role_grants g
          WHERE g.org_id = p_org_id
            AND g.user_id = u.id
            AND g.role = 'org_admin'
            AND g.org_unit_id IS NULL
            AND (g.expires_at IS NULL OR g.expires_at > NOW())
      )
      AND NOT EXISTS (
          SELECT 1 FROM "user".user_app_roles ur
          JOIN "user".app_roles ar ON ar.id = ur.role_id
          WHERE ur.user_id = u.id AND ar.name = 'Global Admin'
      );
$$;