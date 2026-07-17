-- T08: Credential verification & tamper-evidence — verify tokens, revocation, audit log.

ALTER TABLE transcripts.transcript_documents
    ADD COLUMN IF NOT EXISTS verify_token TEXT UNIQUE,
    ADD COLUMN IF NOT EXISTS pdf_hash TEXT,
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS revoke_reason TEXT,
    ADD COLUMN IF NOT EXISTS disclose_publicly BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN transcripts.transcript_documents.verify_token IS
    'High-entropy public token for QR/link verification (T08).';
COMMENT ON COLUMN transcripts.transcript_documents.pdf_hash IS
    'SHA-256 hex of delivered PDF bytes for upload-verify matching (T08).';
COMMENT ON COLUMN transcripts.transcript_documents.revoked_at IS
    'When set, verification returns revoked (T08).';
COMMENT ON COLUMN transcripts.transcript_documents.disclose_publicly IS
    'When true, verify portal may include the full VC; otherwise minimal disclosure (T08).';

CREATE INDEX IF NOT EXISTS idx_transcript_documents_pdf_hash
    ON transcripts.transcript_documents (pdf_hash)
    WHERE pdf_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS transcripts.verifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id   UUID,
    document_type TEXT NOT NULL CHECK (document_type IN ('transcript', 'clr', 'diploma')),
    result        TEXT NOT NULL CHECK (result IN ('genuine', 'tampered', 'revoked', 'not_found')),
    method        TEXT NOT NULL CHECK (method IN ('link', 'qr', 'upload')),
    requester_ip  INET,
    requester_ua  TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.verifications IS
    'Third-party credential verification lookups (T08).';

CREATE INDEX IF NOT EXISTS idx_verifications_doc
    ON transcripts.verifications (document_id, created_at);

CREATE INDEX IF NOT EXISTS idx_verifications_created
    ON transcripts.verifications (created_at DESC);
