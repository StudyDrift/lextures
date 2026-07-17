-- T07: Inbound receiving & transfer-credit intake.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_transcript_inbound BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_transcript_inbound IS
    'T07: Enables inbound transcript receiving, PESC parse/match, and the registrar intake queue.';

CREATE TABLE IF NOT EXISTS transcripts.inbound_documents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES tenant.organizations (id),
    channel          TEXT NOT NULL CHECK (channel IN ('api_peer', 'sftp', 'email')),
    source_name      TEXT,
    external_ref     TEXT,
    format           TEXT NOT NULL CHECK (format IN ('pesc_xml', 'pdf', 'edi', 'other')),
    raw_key          TEXT NOT NULL,
    raw_bytes        BYTEA NOT NULL,
    content_hash     TEXT NOT NULL,
    content_type     TEXT,
    byte_size        INT NOT NULL CHECK (byte_size >= 0),
    parsed           JSONB,
    student_name     TEXT,
    student_dob      TEXT,
    student_ref      TEXT,
    matched_user_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    match_confidence NUMERIC(4, 3),
    match_detail     JSONB,
    status           TEXT NOT NULL DEFAULT 'received'
        CHECK (status IN (
            'received',
            'quarantined',
            'parsed',
            'matched',
            'accepted',
            'rejected',
            'unmatched'
        )),
    needs_manual_review BOOLEAN NOT NULL DEFAULT FALSE,
    reviewer_id      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reject_reason    TEXT,
    quarantine_reason TEXT,
    received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at     TIMESTAMPTZ,
    notified_received_at TIMESTAMPTZ,
    notified_accepted_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.inbound_documents IS
    'Transcripts received from other institutions for transfer-credit intake (T07).';

-- Deduplicate re-sends when sender + external reference are both present.
CREATE UNIQUE INDEX IF NOT EXISTS uq_inbound_dedupe
    ON transcripts.inbound_documents (org_id, source_name, external_ref)
    WHERE source_name IS NOT NULL AND external_ref IS NOT NULL AND BTRIM(source_name) <> '' AND BTRIM(external_ref) <> '';

CREATE INDEX IF NOT EXISTS idx_inbound_queue
    ON transcripts.inbound_documents (org_id, status, received_at DESC);

CREATE INDEX IF NOT EXISTS idx_inbound_matched_user
    ON transcripts.inbound_documents (matched_user_id)
    WHERE matched_user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS transcripts.inbound_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inbound_id  UUID NOT NULL REFERENCES transcripts.inbound_documents (id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,
    actor_id    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    detail      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.inbound_events IS
    'Audit trail for inbound transcript receive/match/accept/reject (T07).';

CREATE INDEX IF NOT EXISTS idx_inbound_events_doc
    ON transcripts.inbound_events (inbound_id, created_at);
