-- Plan B1 — Competency micro-badges (signed, shareable, verifiable Open Badges 3.0).
-- Idempotent CREATE so parallel package migrates / rollback re-apply races stay safe.

CREATE SCHEMA IF NOT EXISTS badges;

CREATE TABLE IF NOT EXISTS badges.badge_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    outcome_id          UUID REFERENCES course.course_learning_outcomes (id) ON DELETE SET NULL,
    sub_outcome_id      UUID REFERENCES course.course_outcome_sub_outcomes (id) ON DELETE SET NULL,
    slug                TEXT NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    criteria_narrative  TEXT NOT NULL DEFAULT '',
    image_key           TEXT,
    tags                TEXT[] NOT NULL DEFAULT '{}',
    alignment_json      JSONB,
    auto_award          BOOLEAN NOT NULL DEFAULT FALSE,
    created_by          UUID NOT NULL REFERENCES "user".users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, slug)
);

COMMENT ON TABLE badges.badge_definitions IS
    'Instructor-defined competency micro-badge Achievement (Open Badges 3.0 BadgeClass; plan B1).';

CREATE INDEX IF NOT EXISTS idx_badge_definitions_course
    ON badges.badge_definitions (course_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_badge_definitions_outcome
    ON badges.badge_definitions (outcome_id)
    WHERE outcome_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS badges.awarded_badges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    definition_id   UUID NOT NULL REFERENCES badges.badge_definitions (id) ON DELETE CASCADE,
    recipient_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    awarded_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    award_source    TEXT NOT NULL DEFAULT 'manual' CHECK (award_source IN ('manual', 'auto')),
    evidence_json   JSONB,
    credential_json JSONB NOT NULL,
    proof           JSONB NOT NULL,
    share_slug      TEXT NOT NULL UNIQUE,
    is_public       BOOLEAN NOT NULL DEFAULT FALSE,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_reason  TEXT,
    revoked_at      TIMESTAMPTZ,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (definition_id, recipient_id)
);

COMMENT ON TABLE badges.awarded_badges IS
    'Issued competency micro-badge assertions with signed OB 3.0 VC proof (plan B1).';

CREATE INDEX IF NOT EXISTS idx_awarded_badges_recipient
    ON badges.awarded_badges (recipient_id, issued_at DESC);

CREATE INDEX IF NOT EXISTS idx_awarded_badges_definition
    ON badges.awarded_badges (definition_id);

CREATE INDEX IF NOT EXISTS idx_awarded_badges_public
    ON badges.awarded_badges (recipient_id, issued_at DESC)
    WHERE is_public AND NOT revoked;

CREATE TABLE IF NOT EXISTS badges.badge_page_views (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    handle_owner_id  UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    awarded_badge_id UUID REFERENCES badges.awarded_badges (id) ON DELETE CASCADE,
    viewed_on        DATE NOT NULL,
    view_count       INT NOT NULL DEFAULT 0,
    UNIQUE (handle_owner_id, awarded_badge_id, viewed_on)
);

COMMENT ON TABLE badges.badge_page_views IS
    'PII-free daily counters for public badge page views (plan B1; mirrors portfolio.portfolio_views).';

CREATE TABLE IF NOT EXISTS badges.reserved_handles (
    handle_lower TEXT PRIMARY KEY
);

COMMENT ON TABLE badges.reserved_handles IS
    'Handles that learners may not claim (routes, brand, abuse prevention; plan B1).';

INSERT INTO badges.reserved_handles (handle_lower) VALUES
    ('admin'),
    ('api'),
    ('verify'),
    ('settings'),
    ('me'),
    ('badges'),
    ('www'),
    ('self'),
    ('support'),
    ('help'),
    ('login'),
    ('signup'),
    ('null'),
    ('undefined'),
    ('lextures'),
    ('system'),
    ('root'),
    ('static'),
    ('assets'),
    ('achievements')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS "user".user_badge_profiles (
    user_id                  UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    handle                   TEXT UNIQUE,
    handle_lower             TEXT UNIQUE GENERATED ALWAYS AS (lower(handle)) STORED,
    page_public              BOOLEAN NOT NULL DEFAULT FALSE,
    search_indexable         BOOLEAN NOT NULL DEFAULT FALSE,
    display_name_override    TEXT,
    hide_real_name           BOOLEAN NOT NULL DEFAULT FALSE,
    handle_changed_at        TIMESTAMPTZ,
    handle_change_count_30d  INT NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_badge_profiles_handle_format CHECK (
        handle IS NULL
        OR handle ~ '^[a-z0-9](?:[a-z0-9-]{1,30})[a-z0-9]$'
    )
);

COMMENT ON TABLE "user".user_badge_profiles IS
    'Learner public badge backpack handle and visibility (plan B1).';

CREATE TABLE IF NOT EXISTS "user".user_badge_handle_history (
    old_handle_lower TEXT PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    released_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE "user".user_badge_handle_history IS
    'Previous badge handles for 301 redirects during grace window (plan B1 FR-13).';

CREATE INDEX IF NOT EXISTS idx_user_badge_handle_history_user
    ON "user".user_badge_handle_history (user_id, released_at DESC);

-- Feature flags (default off).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_competency_badges BOOLEAN;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS badges_default_public BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_competency_badges IS
    'Enables competency micro-badges: define, award, public backpack, verify (plan B1).';

COMMENT ON COLUMN settings.platform_app_settings.badges_default_public IS
    'Tenant default for new badge awards is_public when learner has not overridden (plan B1).';

-- Share / view audit events (extends 283 credential share pattern).
ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_event_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_event_kind_check CHECK (
    event_kind IN (
        'course_visit',
        'content_open',
        'content_leave',
        'equation_inserted',
        'equation_editor_open',
        'credential_share_linkedin',
        'credential_share_badge_export',
        'badge_share_linkedin',
        'badge_share_x',
        'badge_page_view'
    )
);

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind;
ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_structure_item_kind_check CHECK (
    (event_kind = 'course_visit' AND structure_item_id IS NULL)
    OR (event_kind IN ('content_open', 'content_leave') AND structure_item_id IS NOT NULL)
    OR (event_kind IN ('equation_inserted', 'equation_editor_open'))
    OR (
        event_kind IN (
            'credential_share_linkedin',
            'credential_share_badge_export',
            'badge_share_linkedin',
            'badge_share_x',
            'badge_page_view'
        )
        AND structure_item_id IS NOT NULL
    )
);
