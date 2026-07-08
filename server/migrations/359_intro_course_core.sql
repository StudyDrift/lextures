-- IC01 — Intro course foundation: feature flag, system instructor, short_code idempotency key.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS intro_course_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.intro_course_enabled IS
    'Enables the canonical "Welcome to Lextures" intro course and auto-enrollment (IC epic). Default true.';

-- Extend account_type to include non-human system identities (seats, directories, analytics).
ALTER TABLE "user".users DROP CONSTRAINT IF EXISTS users_account_type_check;
ALTER TABLE "user".users
    ADD CONSTRAINT users_account_type_check
        CHECK (account_type IN ('standard', 'parent', 'system'));

-- Mark the platform inbox sender as system (already excluded by UUID in several queries).
UPDATE "user".users
SET account_type = 'system'
WHERE id = 'a0000000-0000-4000-8000-000000000001'::uuid
  AND account_type <> 'system';

-- Dedicated system instructor for the canonical intro course (not intended for login).
INSERT INTO "user".users (id, email, password_hash, display_name, account_type, org_id, login_blocked)
VALUES (
    'a0000000-0000-4000-8000-000000000002'::uuid,
    'guide@system.lextures.invalid',
    '!',
    'Lextures Guide',
    'system',
    (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1),
    TRUE
)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    account_type = EXCLUDED.account_type,
    login_blocked = TRUE;

-- Immutable idempotency key for the canonical intro course (URLs still use course_code).
ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS short_code TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_courses_short_code
    ON course.courses (short_code)
    WHERE short_code IS NOT NULL;

COMMENT ON COLUMN course.courses.short_code IS
    'Optional stable platform key for idempotent provisioning (e.g. LEX-WELCOME intro course).';

-- Exclude system accounts from learner seat counts.
CREATE OR REPLACE FUNCTION tenant.count_learner_seats(p_org_id UUID)
RETURNS INT
LANGUAGE sql
STABLE
AS $$
    SELECT COUNT(*)::INT
    FROM "user".users u
    WHERE u.org_id = p_org_id
      AND u.account_type <> 'system'
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