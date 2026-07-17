-- T12: Registrar console analytics — daily rollups, turnaround view, SLA thresholds, RBAC permissions.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS sla_backlog_threshold INT NOT NULL DEFAULT 25,
    ADD COLUMN IF NOT EXISTS sla_oldest_pending_hours INT NOT NULL DEFAULT 48,
    ADD COLUMN IF NOT EXISTS sla_failure_rate_bps INT NOT NULL DEFAULT 500;

COMMENT ON COLUMN settings.transcripts_config.sla_backlog_threshold IS
    'T12: Alert when open fulfillment queue (in_review/on_hold/processing) exceeds this count.';
COMMENT ON COLUMN settings.transcripts_config.sla_oldest_pending_hours IS
    'T12: Alert when the oldest pending order age exceeds this many hours.';
COMMENT ON COLUMN settings.transcripts_config.sla_failure_rate_bps IS
    'T12: Alert when delivery failure rate (basis points, 500 = 5%) is exceeded in the lookback window.';

CREATE TABLE IF NOT EXISTS transcripts.analytics_daily (
    org_id             UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    day                DATE NOT NULL,
    orders             INT NOT NULL DEFAULT 0,
    items              INT NOT NULL DEFAULT 0,
    delivered          INT NOT NULL DEFAULT 0,
    on_hold            INT NOT NULL DEFAULT 0,
    rejected           INT NOT NULL DEFAULT 0,
    refunded           INT NOT NULL DEFAULT 0,
    net_revenue_minor  BIGINT NOT NULL DEFAULT 0,
    refreshed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, day)
);

CREATE INDEX IF NOT EXISTS idx_transcript_analytics_daily_day
    ON transcripts.analytics_daily (day DESC);

COMMENT ON TABLE transcripts.analytics_daily IS
    'T12: Precomputed daily transcript order/delivery/revenue rollups per org (refreshed on schedule).';

CREATE OR REPLACE VIEW transcripts.v_turnaround AS
SELECT
    oi.order_id,
    oi.id AS item_id,
    o.org_id,
    o.submitted_at,
    COALESCE(
        oi.delivered_at,
        (
            SELECT MIN(da.created_at)
            FROM transcripts.delivery_attempts da
            WHERE da.order_item_id = oi.id
              AND da.status = 'delivered'
        )
    ) AS delivered_at
FROM transcripts.order_items oi
INNER JOIN transcripts.orders o ON o.id = oi.order_id;

COMMENT ON VIEW transcripts.v_turnaround IS
    'T12: Submit → delivered turnaround per order item (authoritative item/attempt timestamps).';

CREATE INDEX IF NOT EXISTS idx_orders_org_created
    ON transcripts.orders (org_id, created_at DESC)
    WHERE org_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_orders_org_status_submitted
    ON transcripts.orders (org_id, status, submitted_at)
    WHERE org_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_order_items_delivery_method
    ON transcripts.order_items (delivery_method);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_status_created
    ON transcripts.delivery_attempts (status, created_at DESC);

-- Panel-scoped permissions (four-segment form). Global Admin retains full access via global:app:rbac:manage.
INSERT INTO "user".permissions (permission_string, description)
VALUES
    ('org:transcripts:console:manage', 'Registrar queue, holds, and fulfillment actions (T12)'),
    ('org:transcripts:finance:view', 'Transcript revenue, holds finance slice, and fee visibility (T12 bursar)'),
    ('org:transcripts:config:manage', 'Transcript fees, delivery, recipients, and console settings (T12 admin)'),
    ('org:transcripts:analytics:view', 'Transcript analytics dashboard (T12)'),
    ('org:transcripts:analytics:export', 'Export transcript analytics CSV reports (T12)')
ON CONFLICT (permission_string) DO NOTHING;

-- Grant new permissions to Global Admin so panel checks succeed alongside global:app:rbac:manage.
INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
CROSS JOIN "user".permissions p
WHERE r.name = 'Global Admin'
  AND p.permission_string IN (
      'org:transcripts:console:manage',
      'org:transcripts:finance:view',
      'org:transcripts:config:manage',
      'org:transcripts:analytics:view',
      'org:transcripts:analytics:export'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
