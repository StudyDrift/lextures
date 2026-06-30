-- Plan 18.8: License / seat management — licenses table, counter, feature flag.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS seat_management_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.seat_management_enabled IS
    'Plan 18.8: Enables org seat license enforcement and admin license management UI.';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type
        WHERE typname = 'license_tier' AND typnamespace = 'tenant'::regnamespace
    ) THEN
        CREATE TYPE tenant.license_tier AS ENUM ('unlimited', 'starter', 'growth', 'enterprise');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS tenant.licenses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL UNIQUE REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    tier            tenant.license_tier NOT NULL DEFAULT 'unlimited',
    max_seats       INT NOT NULL DEFAULT -1,
    used_seats      INT NOT NULL DEFAULT 0,
    contract_start  DATE,
    contract_end    DATE,
    notes           TEXT,
    updated_by      UUID REFERENCES "user".users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT licenses_max_seats_check CHECK (max_seats = -1 OR max_seats >= 0),
    CONSTRAINT licenses_used_seats_nonneg CHECK (used_seats >= 0)
);

COMMENT ON TABLE tenant.licenses IS 'Per-org seat license records (plan 18.8). max_seats = -1 means unlimited.';

CREATE TABLE IF NOT EXISTS tenant.license_utilization_alerts (
    org_id          UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    threshold_pct   INT NOT NULL CHECK (threshold_pct IN (80, 95)),
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, threshold_pct)
);

COMMENT ON TABLE tenant.license_utilization_alerts IS
    'Dedupes seat-utilization alert emails per org and threshold (plan 18.8).';

-- Active learner seats: active users who are not org_admin or Global Admin.
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

CREATE OR REPLACE FUNCTION tenant.refresh_license_used_seats(p_org_id UUID)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO tenant.licenses (org_id, used_seats)
    VALUES (p_org_id, tenant.count_learner_seats(p_org_id))
    ON CONFLICT (org_id) DO UPDATE
        SET used_seats = EXCLUDED.used_seats,
            updated_at = NOW();
END;
$$;

CREATE OR REPLACE FUNCTION tenant.sync_license_on_user_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_org UUID;
BEGIN
    v_org := COALESCE(NEW.org_id, OLD.org_id);
    IF v_org IS NOT NULL THEN
        PERFORM tenant.refresh_license_used_seats(v_org);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS trg_users_license_seat_sync ON "user".users;
CREATE TRIGGER trg_users_license_seat_sync
    AFTER INSERT OR UPDATE OF deactivated_at, login_blocked, org_id OR DELETE
    ON "user".users
    FOR EACH ROW
    EXECUTE FUNCTION tenant.sync_license_on_user_change();

CREATE OR REPLACE FUNCTION tenant.sync_license_on_org_role_grant()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_org UUID;
BEGIN
    v_org := COALESCE(NEW.org_id, OLD.org_id);
    IF v_org IS NOT NULL THEN
        PERFORM tenant.refresh_license_used_seats(v_org);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS trg_org_role_grants_license_seat_sync ON "user".org_role_grants;
CREATE TRIGGER trg_org_role_grants_license_seat_sync
    AFTER INSERT OR UPDATE OR DELETE
    ON "user".org_role_grants
    FOR EACH ROW
    EXECUTE FUNCTION tenant.sync_license_on_org_role_grant();

INSERT INTO settings.email_template_slots (id, description, merge_fields, default_html, default_text)
VALUES (
    'seat_utilization_alert',
    'Seat license utilization threshold alert for org admins (plan 18.8).',
    '{"orgName":"Organization name","usedSeats":"Active learner seats used","maxSeats":"Licensed seat limit","percentUsed":"Utilization percentage","thresholdPct":"Alert threshold percentage","unsubscribeUrl":"One-click unsubscribe link"}'::jsonb,
    '<p>Your organization <strong>{{orgName}}</strong> has reached {{thresholdPct}}% of its licensed seats ({{usedSeats}} / {{maxSeats}}).</p><p>Contact your Lextures representative to purchase additional seats before new users are blocked.</p>',
    'Your organization {{orgName}} has reached {{thresholdPct}}% of its licensed seats ({{usedSeats}} / {{maxSeats}}). Contact your Lextures representative to purchase additional seats.'
)
ON CONFLICT (id) DO NOTHING;
