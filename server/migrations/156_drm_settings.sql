-- DRM / watermarking support for licensed publisher content (plan 8.10).
-- storage schema already exists (migration 153).

CREATE TYPE storage.drm_type AS ENUM ('none', 'watermark_only', 'widevine', 'fairplay');

ALTER TABLE storage.objects
  ADD COLUMN drm_type    storage.drm_type NOT NULL DEFAULT 'none',
  ADD COLUMN drm_key_id  TEXT,
  ADD COLUMN drm_provider TEXT;

CREATE TABLE storage.drm_license_requests (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  object_id      UUID        NOT NULL REFERENCES storage.objects(id) ON DELETE CASCADE,
  user_id        UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  ip_address     INET,
  granted        BOOLEAN     NOT NULL,
  denial_reason  TEXT,
  requested_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON storage.drm_license_requests (object_id, user_id, requested_at);
CREATE INDEX ON storage.drm_license_requests (user_id, object_id, requested_at);
