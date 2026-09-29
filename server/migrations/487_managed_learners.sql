-- Issue #640 — Homeschool parent-managed learners (account_type=managed + feature flag + seat exclusion).

ALTER TABLE "user".users DROP CONSTRAINT IF EXISTS users_account_type_check;
ALTER TABLE "user".users
    ADD CONSTRAINT users_account_type_check
        CHECK (account_type IN ('standard', 'parent', 'system', 'managed'));

COMMENT ON CONSTRAINT users_account_type_check ON "user".users IS
    'standard|parent|system|managed (managed = parent-created learner without real email; #640).';

-- Exclude managed (and system) accounts from learner seat counts.
CREATE OR REPLACE FUNCTION tenant.count_learner_seats(p_org_id UUID)
RETURNS INT
LANGUAGE sql
STABLE
AS $$
    SELECT COUNT(*)::INT
    FROM "user".users u
    WHERE u.org_id = p_org_id
      AND u.account_type NOT IN ('system', 'managed')
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

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_homeschool_managed_learners BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_homeschool_managed_learners IS
    'Enables parent-managed learners (Homeschool #640): create dependents, enroll by learnerUserIds, Learn-as. Default ON.';
