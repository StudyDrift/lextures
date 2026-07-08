-- LP09 — Profile-powered adaptivity consumer sub-flags.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS lp_adapt_recommendations BOOLEAN,
    ADD COLUMN IF NOT EXISTS lp_adapt_review BOOLEAN,
    ADD COLUMN IF NOT EXISTS lp_adapt_modality BOOLEAN,
    ADD COLUMN IF NOT EXISTS lp_adapt_tutor BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.lp_adapt_recommendations IS
    'When true with learner_profile_enabled, recommendations incorporate profile interests and growth areas (LP09).';
COMMENT ON COLUMN settings.platform_app_settings.lp_adapt_review IS
    'When true with learner_profile_enabled, SRS review queue prioritises needs-review concepts and peak study windows (LP09).';
COMMENT ON COLUMN settings.platform_app_settings.lp_adapt_modality IS
    'When true with learner_profile_enabled, content selection prefers the learner''s preferred modality when alternates exist (LP09).';
COMMENT ON COLUMN settings.platform_app_settings.lp_adapt_tutor IS
    'When true with learner_profile_enabled, persistent tutor adjusts scaffolding to help-seeking style (LP09).';