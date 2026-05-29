-- Plan 12.4: Captions on uploaded media (WCAG 1.2.2 UI/workflow layer on storage.captions).

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS require_captions BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.require_captions IS
    'When true, publishing module items with embedded video requires ready captions (plan 12.4).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS video_captions_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.video_captions_enabled IS
    'Plan 12.4 (ff_video_captions): caption editor, player controls, compliance report, and import/export.';

-- Backfill queued caption jobs for existing videos without a caption row.
INSERT INTO storage.captions (storage_object_id, backend, status)
SELECT o.id, 'import', 'queued'
FROM storage.objects o
WHERE o.deleted_at IS NULL
  AND o.mime_type LIKE 'video/%'
  AND NOT EXISTS (
      SELECT 1 FROM storage.captions c WHERE c.storage_object_id = o.id
  );
