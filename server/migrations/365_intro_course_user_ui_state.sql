-- IC06: per-user intro course onboarding UI state (welcome banner + completion celebration).

CREATE TABLE IF NOT EXISTS settings.intro_course_user_ui_state (
    user_id                      UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    welcome_banner_dismissed_at  TIMESTAMPTZ,
    celebration_seen_at          TIMESTAMPTZ,
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE settings.intro_course_user_ui_state IS
    'Per-learner intro course onboarding surface state (IC06). Progress lives in derived grades; this stores dismiss/seen flags.';