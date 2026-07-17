-- T09: Learner credential wallet — unified index, collections, share links, exports.

CREATE TABLE IF NOT EXISTS credentials.wallet_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    kind          TEXT NOT NULL CHECK (kind IN (
        'transcript', 'clr', 'badge', 'certificate', 'diploma', 'ce_record'
    )),
    source_id     UUID NOT NULL,
    title         TEXT NOT NULL,
    issuer        TEXT,
    issued_at     TIMESTAMPTZ,
    verify_token  TEXT,
    revoked       BOOLEAN NOT NULL DEFAULT FALSE,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, kind, source_id)
);

COMMENT ON TABLE credentials.wallet_items IS
    'Unified learner credential index refreshed from source tables (T09).';

CREATE INDEX IF NOT EXISTS idx_wallet_items_user_issued
    ON credentials.wallet_items (user_id, issued_at DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS credentials.collections (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    share_token  TEXT UNIQUE,
    disclosure   TEXT NOT NULL DEFAULT 'validity'
        CHECK (disclosure IN ('validity', 'summary', 'full')),
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE credentials.collections IS
    'Learner-curated credential collections with shareable verifiable links (T09).';

CREATE INDEX IF NOT EXISTS idx_collections_user
    ON credentials.collections (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS credentials.collection_items (
    collection_id  UUID NOT NULL REFERENCES credentials.collections (id) ON DELETE CASCADE,
    wallet_item_id UUID NOT NULL REFERENCES credentials.wallet_items (id) ON DELETE CASCADE,
    position       INT NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_id, wallet_item_id)
);

CREATE TABLE IF NOT EXISTS credentials.collection_access (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id  UUID NOT NULL REFERENCES credentials.collections (id) ON DELETE CASCADE,
    result         TEXT NOT NULL CHECK (result IN ('ok', 'revoked', 'expired', 'not_found')),
    requester_ip   INET,
    requester_ua   TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE credentials.collection_access IS
    'Access history for shared credential collections (T09).';

CREATE INDEX IF NOT EXISTS idx_collection_access_collection
    ON credentials.collection_access (collection_id, created_at DESC);

CREATE TABLE IF NOT EXISTS credentials.wallet_exports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'ready', 'failed')),
    zip_bytes     BYTEA,
    manifest      JSONB,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

COMMENT ON TABLE credentials.wallet_exports IS
    'Async portable wallet export bundles (ZIP of PDFs + VC JSON + manifest) (T09).';

CREATE INDEX IF NOT EXISTS idx_wallet_exports_user
    ON credentials.wallet_exports (user_id, created_at DESC);
