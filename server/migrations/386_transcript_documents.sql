-- T01: Immutable issued academic transcript documents (canonical JSON + PDF + PESC XML).

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS official_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.transcripts_config.official_enabled IS
    'When true (and ff_transcripts), students/registrars may issue official sealed transcripts. Unofficial preview remains available when ff_transcripts is on.';

CREATE TABLE IF NOT EXISTS transcripts.transcript_documents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id           UUID REFERENCES tenant.organizations (id),
    variant          TEXT NOT NULL
                       CHECK (variant IN ('official', 'unofficial', 'partial', 'in_progress')),
    version          INT  NOT NULL,
    canonical        TEXT NOT NULL, -- byte-stable JSON (not JSONB) so content_hash verifies exact bytes
    schema_version   TEXT NOT NULL,
    template_version TEXT NOT NULL,
    content_hash     TEXT NOT NULL,
    pdf_bytes        BYTEA,
    pesc_xml_bytes   BYTEA,
    pdf_key          TEXT,
    pesc_xml_key     TEXT,
    vc_proof         JSONB,
    gpa_cumulative   NUMERIC(4, 3),
    credits_earned   NUMERIC(7, 2),
    generated_by     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    generated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transcript_documents_version_positive CHECK (version > 0)
);

COMMENT ON TABLE transcripts.transcript_documents IS
    'Immutable issued academic-record artifacts. Re-issuance creates a new version; prior rows are never updated.';

COMMENT ON COLUMN transcripts.transcript_documents.content_hash IS
    'SHA-256 hex of the canonical JSON bytes (byte-stable serialization).';

COMMENT ON COLUMN transcripts.transcript_documents.pdf_bytes IS
    'Cached PDF bytes when object storage is unavailable; prefer pdf_key when set.';

CREATE UNIQUE INDEX IF NOT EXISTS ux_transcript_documents_official_version
    ON transcripts.transcript_documents (user_id, version)
    WHERE variant = 'official';

CREATE INDEX IF NOT EXISTS idx_transcript_documents_user
    ON transcripts.transcript_documents (user_id, generated_at DESC);

CREATE INDEX IF NOT EXISTS idx_transcript_documents_org
    ON transcripts.transcript_documents (org_id, generated_at DESC)
    WHERE org_id IS NOT NULL;
