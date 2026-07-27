-- CT.9 — Content Tools marketplace & third-party tools.

CREATE SCHEMA IF NOT EXISTS toolmarket;

-- A publishable tool owned by a developer (person or org).
CREATE TABLE IF NOT EXISTS toolmarket.tools (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id           TEXT NOT NULL UNIQUE,      -- global namespace, immutable, e.g. 'acme.titration_lab'
    owner_user_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    owner_org_id      UUID REFERENCES tenant.organizations (id) ON DELETE SET NULL,
    display_name      TEXT NOT NULL,
    summary           TEXT NOT NULL,
    description_md    TEXT NOT NULL DEFAULT '',
    subject_tags      TEXT[] NOT NULL DEFAULT '{}',
    grade_tags        TEXT[] NOT NULL DEFAULT '{}',
    support_url       TEXT,
    privacy_url       TEXT,
    visibility        TEXT NOT NULL DEFAULT 'private'
                        CHECK (visibility IN ('private','unlisted','public')),
    pricing_model     TEXT NOT NULL DEFAULT 'free'
                        CHECK (pricing_model IN ('free','paid','trial')),
    status            TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','in_review','approved','rejected','suspended','sunset')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tm_tools_status ON toolmarket.tools (status, visibility);
CREATE INDEX IF NOT EXISTS idx_tm_tools_owner ON toolmarket.tools (owner_user_id);

COMMENT ON TABLE toolmarket.tools IS
    'CT.9: Third-party Content Tools catalog entries owned by developers.';

-- Immutable published versions with their bundle + manifest + data sheet.
CREATE TABLE IF NOT EXISTS toolmarket.tool_releases (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_pk           UUID NOT NULL REFERENCES toolmarket.tools (id) ON DELETE CASCADE,
    version           TEXT NOT NULL,
    manifest_json     JSONB NOT NULL,
    data_sheet_json   JSONB NOT NULL,
    bundle_object_id  UUID REFERENCES storage.objects (id) ON DELETE SET NULL,
    bundle_sri        TEXT NOT NULL,
    bundle_bytes      INTEGER NOT NULL DEFAULT 0,
    checks_json       JSONB NOT NULL DEFAULT '{}'::jsonb,   -- automated results
    review_status     TEXT NOT NULL DEFAULT 'pending'
                        CHECK (review_status IN ('pending','approved','rejected')),
    reviewed_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    review_notes      TEXT,
    published_at      TIMESTAMPTZ,
    sunset_at         TIMESTAMPTZ,
    soak_until        TIMESTAMPTZ, -- for minor/patch auto-update window
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tool_pk, version)
);
CREATE INDEX IF NOT EXISTS idx_tmr_review ON toolmarket.tool_releases (review_status, created_at);

COMMENT ON TABLE toolmarket.tool_releases IS
    'CT.9: Immutable Content Tool releases with SRI, checks, and review state.';

-- Org installations, with the consented capability + host set frozen at consent time.
CREATE TABLE IF NOT EXISTS toolmarket.tool_installations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    tool_pk           UUID NOT NULL REFERENCES toolmarket.tools (id) ON DELETE CASCADE,
    pinned_major      INTEGER NOT NULL,
    current_version   TEXT NOT NULL,
    consented_capabilities TEXT[] NOT NULL DEFAULT '{}',
    consented_hosts   TEXT[] NOT NULL DEFAULT '{}',
    auto_update_minor BOOLEAN NOT NULL DEFAULT TRUE,
    status            TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','revoked','suspended')),
    installed_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    installed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at        TIMESTAMPTZ,
    UNIQUE (org_id, tool_pk)
);
CREATE INDEX IF NOT EXISTS idx_tmi_org_status ON toolmarket.tool_installations (org_id, status);

COMMENT ON TABLE toolmarket.tool_installations IS
    'CT.9: Org-scoped Content Tool installations with frozen consent.';

-- Invitations for unlisted distribution.
CREATE TABLE IF NOT EXISTS toolmarket.tool_access_grants (
    tool_pk   UUID NOT NULL REFERENCES toolmarket.tools (id) ON DELETE CASCADE,
    org_id    UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    granted_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tool_pk, org_id)
);

COMMENT ON TABLE toolmarket.tool_access_grants IS
    'CT.9: Access grants for private/unlisted Content Tools.';

-- Lifecycle audit mirror (also written to adminaudit).
CREATE TABLE IF NOT EXISTS toolmarket.tool_lifecycle_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_pk     UUID REFERENCES toolmarket.tools (id) ON DELETE SET NULL,
    release_id  UUID REFERENCES toolmarket.tool_releases (id) ON DELETE SET NULL,
    org_id      UUID REFERENCES tenant.organizations (id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    details_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tm_lifecycle ON toolmarket.tool_lifecycle_events (tool_pk, created_at DESC);

COMMENT ON TABLE toolmarket.tool_lifecycle_events IS
    'CT.9: Lifecycle audit trail for marketplace actions.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_content_tool_marketplace BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_content_tool_marketplace IS
    'CT.9: Enables the Content Tools third-party marketplace (default false).';
