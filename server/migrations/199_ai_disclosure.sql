-- 10.17 AI Usage Disclosure: opt-out, inference audit log, tenant AI governance (plan 10.17).
-- Depends on: compliance schema (179_ferpa), "user".users (011), tenant.organizations (127).

ALTER TABLE "user".users
  ADD COLUMN IF NOT EXISTS ai_processing_opt_out BOOLEAN NOT NULL DEFAULT FALSE;

-- COPPA minors default to opted out until parental AI opt-in (FR-6, backward compat).
UPDATE "user".users
   SET ai_processing_opt_out = TRUE
 WHERE coppa_minor = TRUE
   AND ai_processing_opt_out = FALSE;

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log (
  id               UUID        NOT NULL DEFAULT gen_random_uuid(),
  org_id           UUID        REFERENCES tenant.organizations(id),
  user_id_hash     TEXT        NOT NULL,
  feature_name     TEXT        NOT NULL,
  model_id         TEXT        NOT NULL,
  provider         TEXT        NOT NULL,
  content_hash     TEXT        NOT NULL,
  opt_in_confirmed BOOLEAN     NOT NULL,
  blocked          BOOLEAN     NOT NULL DEFAULT FALSE,
  "timestamp"      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id, "timestamp")
) PARTITION BY RANGE ("timestamp");

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_05
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_06
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_07
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_08
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_09
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_10
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_11
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_2026_12
  PARTITION OF compliance.ai_inference_log
  FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

CREATE TABLE IF NOT EXISTS compliance.ai_inference_log_overflow
  PARTITION OF compliance.ai_inference_log DEFAULT;

CREATE INDEX IF NOT EXISTS idx_ai_inference_log_user
  ON compliance.ai_inference_log(user_id_hash, "timestamp" DESC);

CREATE INDEX IF NOT EXISTS idx_ai_inference_log_org_ts
  ON compliance.ai_inference_log(org_id, "timestamp" DESC);

COMMENT ON TABLE compliance.ai_inference_log IS
  'Append-only AI inference audit trail; FERPA disclosure support; plan 10.17 FR-5.';

CREATE TABLE IF NOT EXISTS compliance.tenant_ai_config (
  org_id           UUID PRIMARY KEY REFERENCES tenant.organizations(id) ON DELETE CASCADE,
  features_enabled JSONB       NOT NULL DEFAULT '{}',
  allowed_models   TEXT[],
  updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_by       UUID         REFERENCES "user".users(id)
);

COMMENT ON TABLE compliance.tenant_ai_config IS
  'Per-tenant AI feature and model governance; plan 10.17 FR-4.';

CREATE TABLE IF NOT EXISTS settings.user_ai_feature_acknowledgements (
  user_id         UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  feature_key     TEXT        NOT NULL,
  acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, feature_key)
);

COMMENT ON TABLE settings.user_ai_feature_acknowledgements IS
  'First-use AI disclosure acknowledgements per feature; plan 10.17 FR-2.';

INSERT INTO "user".permissions (permission_string, description)
VALUES ('compliance:ai:read:*', 'May query the AI inference compliance log (plan 10.17).')
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM "user".app_roles r
  JOIN "user".permissions p ON p.permission_string = 'compliance:ai:read:*'
 WHERE r.name = 'Global Admin'
ON CONFLICT DO NOTHING;

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS ai_disclosure_enabled BOOLEAN;
