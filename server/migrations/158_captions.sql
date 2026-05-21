-- Auto-captioning & transcripts for video content (plan 8.4).
-- storage schema already exists (migration 153).

CREATE TYPE storage.caption_status AS ENUM
  ('queued', 'processing', 'done', 'failed', 'api_unavailable', 'instructor_reviewed');

CREATE TABLE storage.captions (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  storage_object_id   UUID        NOT NULL REFERENCES storage.objects(id) ON DELETE CASCADE,
  lang                TEXT        NOT NULL DEFAULT 'en',
  vtt_key             TEXT,
  transcript_text     TEXT,
  confidence_avg      REAL,
  backend             TEXT        NOT NULL,
  status              storage.caption_status NOT NULL DEFAULT 'queued',
  has_low_confidence  BOOLEAN     NOT NULL DEFAULT false,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  reviewed_at         TIMESTAMPTZ,
  reviewed_by         UUID        REFERENCES "user".users(id) ON DELETE SET NULL
);

CREATE INDEX ON storage.captions (storage_object_id, lang);
CREATE INDEX ON storage.captions (status) WHERE status NOT IN ('done', 'instructor_reviewed');
