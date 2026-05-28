-- 10.15 Backup / Restore / RPO-RTO: restore drill records, backup tier status, feature flag.
-- Depends on: compliance schema (179_ferpa), "user".users (011).

CREATE TABLE IF NOT EXISTS compliance.restore_drills (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  drill_date           DATE NOT NULL,
  backup_timestamp     TIMESTAMPTZ NOT NULL,
  restore_start        TIMESTAMPTZ NOT NULL,
  restore_end          TIMESTAMPTZ,
  rpo_achieved_minutes INTEGER,
  rto_achieved_minutes INTEGER,
  pass                 BOOLEAN,
  smoke_test_output    TEXT,
  conducted_by         UUID REFERENCES "user".users(id) ON DELETE SET NULL,
  notes                TEXT,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_restore_drills_drill_date
  ON compliance.restore_drills(drill_date DESC);

COMMENT ON TABLE compliance.restore_drills IS
  'Quarterly restore drill results (plan 10.15 FR-7, AC-3).';

-- Last-known backup health per data tier (updated by WAL-G cron / backup-report CLI).
CREATE TABLE IF NOT EXISTS compliance.backup_tier_status (
  tier                  TEXT PRIMARY KEY CHECK (tier IN ('postgres', 'object_storage')),
  last_success_at       TIMESTAMPTZ,
  last_duration_seconds INTEGER,
  wal_lag_seconds       INTEGER,
  next_scheduled_at     TIMESTAMPTZ,
  last_error            TEXT,
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO compliance.backup_tier_status (tier)
VALUES ('postgres'), ('object_storage')
ON CONFLICT (tier) DO NOTHING;

COMMENT ON TABLE compliance.backup_tier_status IS
  'Backup job heartbeats for ops dashboard (plan 10.15 FR-1, FR-3, observability).';

INSERT INTO "user".permissions (permission_string, description)
VALUES (
  'compliance:backup:admin:*',
  'May access backup/restore ops: backup status and restore drill records (plan 10.15).'
)
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM "user".app_roles r
  JOIN "user".permissions p ON p.permission_string = 'compliance:backup:admin:*'
 WHERE r.name = 'Global Admin'
ON CONFLICT DO NOTHING;

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS backup_module_enabled BOOLEAN;
