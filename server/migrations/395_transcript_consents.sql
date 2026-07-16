-- T04: FERPA release authorizations (e-signature), consent_required flag, orders.consent_id FK.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS consent_required BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN settings.transcripts_config.consent_required IS
    'When true (default), third-party transcript releases require a signed FERPA authorization before leaving pending_consent.';

CREATE TABLE IF NOT EXISTS transcripts.consents (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id              UUID NOT NULL REFERENCES transcripts.orders (id) ON DELETE CASCADE,
    user_id               UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    signer_id             UUID NOT NULL REFERENCES "user".users (id),
    signer_role           TEXT NOT NULL CHECK (signer_role IN ('student', 'guardian')),
    guardian_relationship TEXT,
    recipients            JSONB NOT NULL,
    scope                 TEXT NOT NULL DEFAULT 'full_academic_record',
    purpose               TEXT,
    text_version          TEXT NOT NULL,
    locale                TEXT NOT NULL DEFAULT 'en',
    signature_method      TEXT NOT NULL CHECK (signature_method IN ('typed', 'drawn')),
    signature_data        TEXT,
    signed_ip             INET,
    signed_ua             TEXT,
    payload_hash          TEXT NOT NULL,
    signed_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at            TIMESTAMPTZ,
    expires_at            TIMESTAMPTZ
);

COMMENT ON TABLE transcripts.consents IS
    'Append-only FERPA §99.30 release authorizations for transcript orders; revocation sets revoked_at only.';

COMMENT ON COLUMN transcripts.consents.recipients IS
    'Snapshot of authorized recipients (id, type, name) at signing time.';

COMMENT ON COLUMN transcripts.consents.payload_hash IS
    'SHA-256 hex of the canonical authorization payload (tamper-evident).';

COMMENT ON COLUMN transcripts.consents.text_version IS
    'Version id of the authorization text signed (e.g. ferpa-release-v1).';

CREATE INDEX IF NOT EXISTS idx_consents_order
    ON transcripts.consents (order_id);

CREATE INDEX IF NOT EXISTS idx_consents_user
    ON transcripts.consents (user_id, signed_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_consents_order_active
    ON transcripts.consents (order_id)
    WHERE revoked_at IS NULL;

-- Deferred FK from T02: orders.consent_id → consents(id).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'orders_consent_id_fkey'
    ) THEN
        ALTER TABLE transcripts.orders
            ADD CONSTRAINT orders_consent_id_fkey
            FOREIGN KEY (consent_id) REFERENCES transcripts.consents (id)
            ON DELETE SET NULL;
    END IF;
END $$;
