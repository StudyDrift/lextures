-- Video transcoding jobs for HLS/DASH adaptive streaming (plan 8.3).
-- storage schema already exists (migration 153).

CREATE TYPE storage.transcode_status AS ENUM ('queued', 'processing', 'done', 'failed');

CREATE TABLE storage.transcode_jobs (
  id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  source_key        TEXT          NOT NULL,
  output_prefix     TEXT,
  master_playlist   TEXT,
  dash_manifest     TEXT,
  poster_key        TEXT,
  status            storage.transcode_status NOT NULL DEFAULT 'queued',
  attempts          SMALLINT      NOT NULL DEFAULT 0,
  error             TEXT,
  created_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
  started_at        TIMESTAMPTZ,
  completed_at      TIMESTAMPTZ,
  storage_object_id UUID          REFERENCES storage.objects(id) ON DELETE SET NULL
);

CREATE INDEX ON storage.transcode_jobs (status, created_at);
CREATE INDEX ON storage.transcode_jobs (source_key);
