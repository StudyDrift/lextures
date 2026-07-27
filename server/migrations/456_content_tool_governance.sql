-- CT.8 — Content Tools governance: org policy, moderation, AI consent, data sheets, incident kills.

-- Org-level policy over tools and tool capabilities.
CREATE TABLE IF NOT EXISTS tenant.content_tool_policies (
    org_id              UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    denied_capabilities TEXT[] NOT NULL DEFAULT '{}',   -- ai|peer_visible|network|media_capture|code_execution
    denied_tool_ids     TEXT[] NOT NULL DEFAULT '{}',
    allowed_tool_ids    TEXT[] NOT NULL DEFAULT '{}',   -- non-empty = strict allowlist
    ai_disclosure_mode  TEXT NOT NULL DEFAULT 'banner'
                          CHECK (ai_disclosure_mode IN ('none','banner','acknowledge')),
    free_text_filter_action TEXT NOT NULL DEFAULT 'flag'
                          CHECK (free_text_filter_action IN ('allow','flag','block')),
    crisis_escalation_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ai_log_retention_days INTEGER NOT NULL DEFAULT 30,
    updated_by          UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE tenant.content_tool_policies IS
    'CT.8: Org-level Content Tools capability/tool policy and safety settings.';

-- Moderation actions on peer-visible tool content.
CREATE TABLE IF NOT EXISTS course.content_tool_moderation (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id   UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    state_id      UUID REFERENCES course.content_tool_states (id) ON DELETE CASCADE,
    content_path  TEXT,                     -- JSON pointer into state_json (e.g. /posts/2)
    action        TEXT NOT NULL CHECK (action IN ('reported','hidden','removed','restored','warned')),
    category      TEXT,                     -- abuse|harassment|off_topic|self_harm|other
    reason        TEXT,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    subject_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ctm_instance ON course.content_tool_moderation (instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctm_subject ON course.content_tool_moderation (subject_user_id, created_at DESC);

COMMENT ON TABLE course.content_tool_moderation IS
    'CT.8: Peer-visible content tool moderation / report audit trail.';

-- Per-student AI disclosure acknowledgements / opt-outs for tools.
CREATE TABLE IF NOT EXISTS course.content_tool_ai_consents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id     UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    tool_id       TEXT,
    decision      TEXT NOT NULL CHECK (decision IN ('acknowledged','opted_out')),
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, course_id, tool_id)
);
CREATE INDEX IF NOT EXISTS idx_ctac_user ON course.content_tool_ai_consents (user_id);

COMMENT ON TABLE course.content_tool_ai_consents IS
    'CT.8: Per-user AI disclosure acknowledgement / opt-out for Content Tools.';

-- Registry mirror of the declarative Tool Data Sheet (for the trust centre + audits).
CREATE TABLE IF NOT EXISTS course.content_tool_data_sheets (
    tool_id         TEXT PRIMARY KEY,
    version         TEXT NOT NULL,
    collects_json   JSONB NOT NULL,     -- field → purpose → retention
    leaves_platform BOOLEAN NOT NULL DEFAULT FALSE,
    processors      TEXT[] NOT NULL DEFAULT '{}',
    visibility      TEXT NOT NULL CHECK (visibility IN ('self','instructor','peers','public')),
    wcag_level      TEXT NOT NULL DEFAULT 'AA',
    a11y_limitations TEXT,
    ai_transparency_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.content_tool_data_sheets IS
    'CT.8: Trust-centre data sheets synced from the tool registry on boot.';

-- Durable incident kill path (tool / capability / all_ai / instance).
CREATE TABLE IF NOT EXISTS settings.content_tool_kills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       TEXT NOT NULL CHECK (scope IN ('tool','capability','all_ai','instance')),
    target      TEXT NOT NULL DEFAULT '',
    engaged     BOOLEAN NOT NULL DEFAULT TRUE,
    reason      TEXT,
    updated_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ctk_scope_target
    ON settings.content_tool_kills (scope, target)
    WHERE engaged = TRUE;

COMMENT ON TABLE settings.content_tool_kills IS
    'CT.8: Durable incident kill path for Content Tools (no deploy required).';

-- Content-filter aggregate flags (blocked raw text is not stored).
CREATE TABLE IF NOT EXISTS course.content_tool_filter_flags (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id   UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    user_id       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    category      TEXT NOT NULL, -- profanity|crisis|policy
    action        TEXT NOT NULL CHECK (action IN ('flag','block')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ctff_instance ON course.content_tool_filter_flags (instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctff_course ON course.content_tool_filter_flags (course_id, created_at DESC);

COMMENT ON TABLE course.content_tool_filter_flags IS
    'CT.8: Aggregate content-filter hits without storing blocked raw student text.';
