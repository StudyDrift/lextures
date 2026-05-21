-- Storage quota settings and usage counters (plan 8.5)

CREATE TABLE storage.quota_settings (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scope       TEXT NOT NULL CHECK (scope IN ('tenant', 'course', 'user')),
  scope_id    UUID NOT NULL,
  limit_bytes BIGINT,                        -- NULL = unlimited
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (scope, scope_id)
);

CREATE TABLE storage.usage_counters (
  scope       TEXT NOT NULL CHECK (scope IN ('tenant', 'course', 'user')),
  scope_id    UUID NOT NULL,
  used_bytes  BIGINT NOT NULL DEFAULT 0 CHECK (used_bytes >= 0),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (scope, scope_id)
);
