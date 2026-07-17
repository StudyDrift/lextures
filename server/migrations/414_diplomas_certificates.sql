-- T11: Diploma & digital certificate issuance.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_diplomas BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_diplomas IS
    'T11: Enables registrar diploma/certificate templates, issuance, wallet surfacing, and verification.';

-- Named diploma_templates to avoid colliding with credentials.credential_templates (plan 15.5).
CREATE TABLE IF NOT EXISTS credentials.diploma_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('diploma', 'certificate')),
    name        TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    program     TEXT,
    conferral_text TEXT,
    layout      JSONB NOT NULL DEFAULT '{}'::jsonb,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE credentials.diploma_templates IS
    'T11 registrar diploma/certificate templates (layout, seal/signature asset keys, conferral text).';

CREATE INDEX IF NOT EXISTS idx_diploma_templates_org
    ON credentials.diploma_templates (org_id, active, created_at DESC);

CREATE TABLE IF NOT EXISTS credentials.diplomas (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id           UUID NOT NULL REFERENCES tenant.organizations (id),
    template_id      UUID REFERENCES credentials.diploma_templates (id) ON DELETE SET NULL,
    kind             TEXT NOT NULL CHECK (kind IN ('diploma', 'certificate')),
    credential_title TEXT NOT NULL,
    program          TEXT,
    honors           TEXT,
    conferred_at     TIMESTAMPTZ NOT NULL,
    version          INT NOT NULL DEFAULT 1,
    replaces_id      UUID REFERENCES credentials.diplomas (id) ON DELETE SET NULL,
    canonical        JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash     TEXT NOT NULL,
    pdf_bytes        BYTEA,
    pdf_key          TEXT,
    vc_proof         JSONB,
    verify_token     TEXT UNIQUE,
    revoked_at       TIMESTAMPTZ,
    revoke_reason    TEXT,
    issued_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    issued_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    program_ref      UUID,
    UNIQUE NULLS NOT DISTINCT (user_id, template_id, program_ref)
);

COMMENT ON TABLE credentials.diplomas IS
    'T11 issued diplomas and formal certificates (signed PDF + VC, wallet + verify).';

CREATE INDEX IF NOT EXISTS idx_diplomas_user
    ON credentials.diplomas (user_id, issued_at DESC);

CREATE INDEX IF NOT EXISTS idx_diplomas_org
    ON credentials.diplomas (org_id, issued_at DESC);

CREATE INDEX IF NOT EXISTS idx_diplomas_verify_token
    ON credentials.diplomas (verify_token)
    WHERE verify_token IS NOT NULL;

CREATE TABLE IF NOT EXISTS credentials.diploma_batches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    template_id   UUID NOT NULL REFERENCES credentials.diploma_templates (id) ON DELETE CASCADE,
    program_ref   UUID,
    program       TEXT,
    honors        TEXT,
    conferred_at  TIMESTAMPTZ NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    total_count   INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    fail_count    INT NOT NULL DEFAULT 0,
    skip_count    INT NOT NULL DEFAULT 0,
    error_summary TEXT,
    created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ
);

COMMENT ON TABLE credentials.diploma_batches IS
    'T11 cohort batch issuance jobs (resumable, idempotent per learner).';

CREATE TABLE IF NOT EXISTS credentials.diploma_batch_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id    UUID NOT NULL REFERENCES credentials.diploma_batches (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    diploma_id  UUID REFERENCES credentials.diplomas (id) ON DELETE SET NULL,
    status      TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'issued', 'skipped', 'failed')),
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (batch_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_diploma_batch_items_batch
    ON credentials.diploma_batch_items (batch_id, status);
