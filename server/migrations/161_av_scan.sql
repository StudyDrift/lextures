-- Antivirus / malware scanning (plan 8.6). Scan metadata on storage.objects; jobs in av_scan_jobs.

CREATE TYPE storage.scan_status AS ENUM ('pending', 'clean', 'quarantined', 'scan_error');

ALTER TABLE storage.objects
  ADD COLUMN IF NOT EXISTS scan_status storage.scan_status NOT NULL DEFAULT 'clean',
  ADD COLUMN IF NOT EXISTS scan_completed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS virus_name TEXT,
  ADD COLUMN IF NOT EXISTS scan_attempts SMALLINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS storage_objects_scan_pending_idx
  ON storage.objects (scan_status) WHERE scan_status = 'pending' AND deleted_at IS NULL;

CREATE TYPE storage.av_scan_job_status AS ENUM ('queued', 'processing', 'done', 'failed');

CREATE TABLE storage.av_scan_jobs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storage_object_id UUID NOT NULL REFERENCES storage.objects(id) ON DELETE CASCADE,
  status            storage.av_scan_job_status NOT NULL DEFAULT 'queued',
  attempts          SMALLINT NOT NULL DEFAULT 0,
  error             TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at        TIMESTAMPTZ,
  completed_at      TIMESTAMPTZ
);

CREATE INDEX ON storage.av_scan_jobs (status, created_at);
CREATE INDEX ON storage.av_scan_jobs (storage_object_id);
