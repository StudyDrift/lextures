-- Reverse T12 transcript analytics.

DELETE FROM "user".rbac_role_permissions rp
USING "user".app_roles r, "user".permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.name = 'Global Admin'
  AND p.permission_string IN (
      'org:transcripts:console:manage',
      'org:transcripts:finance:view',
      'org:transcripts:config:manage',
      'org:transcripts:analytics:view',
      'org:transcripts:analytics:export'
  );

DELETE FROM "user".permissions
WHERE permission_string IN (
    'org:transcripts:console:manage',
    'org:transcripts:finance:view',
    'org:transcripts:config:manage',
    'org:transcripts:analytics:view',
    'org:transcripts:analytics:export'
);

DROP INDEX IF EXISTS transcripts.idx_delivery_attempts_status_created;
DROP INDEX IF EXISTS transcripts.idx_order_items_delivery_method;
DROP INDEX IF EXISTS transcripts.idx_orders_org_status_submitted;
DROP INDEX IF EXISTS transcripts.idx_orders_org_created;

DROP VIEW IF EXISTS transcripts.v_turnaround;
DROP TABLE IF EXISTS transcripts.analytics_daily;

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS sla_failure_rate_bps,
    DROP COLUMN IF EXISTS sla_oldest_pending_hours,
    DROP COLUMN IF EXISTS sla_backlog_threshold;
