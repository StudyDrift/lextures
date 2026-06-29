-- 21.5 / M0.1 — Native device push tokens (APNs + FCM), separate from web push subscriptions.

CREATE TABLE IF NOT EXISTS settings.device_push_tokens (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  token         TEXT NOT NULL,
  platform      TEXT NOT NULL CHECK (platform IN ('apns', 'fcm')),
  app_bundle_id TEXT,
  app_version   TEXT,
  is_active     BOOLEAN NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at  TIMESTAMPTZ,
  UNIQUE (user_id, token)
);

CREATE INDEX IF NOT EXISTS idx_device_push_tokens_user_active
  ON settings.device_push_tokens (user_id, is_active);
