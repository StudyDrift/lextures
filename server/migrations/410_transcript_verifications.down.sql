DROP INDEX IF EXISTS transcripts.idx_verifications_created;
DROP INDEX IF EXISTS transcripts.idx_verifications_doc;
DROP TABLE IF EXISTS transcripts.verifications;

DROP INDEX IF EXISTS transcripts.idx_transcript_documents_pdf_hash;

ALTER TABLE transcripts.transcript_documents
    DROP COLUMN IF EXISTS disclose_publicly,
    DROP COLUMN IF EXISTS revoke_reason,
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS pdf_hash,
    DROP COLUMN IF EXISTS verify_token;
