DROP INDEX IF EXISTS transcripts.idx_transcript_documents_org;
DROP INDEX IF EXISTS transcripts.idx_transcript_documents_user;
DROP INDEX IF EXISTS transcripts.ux_transcript_documents_official_version;
DROP TABLE IF EXISTS transcripts.transcript_documents;
ALTER TABLE settings.transcripts_config DROP COLUMN IF EXISTS official_enabled;
