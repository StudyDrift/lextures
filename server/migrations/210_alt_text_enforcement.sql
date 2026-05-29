-- Plan 12.5: Alt-text enforcement in content authoring (WCAG SC 1.1.1).
-- alt_text_enforcement_enabled: master platform toggle (Settings → Global platform).
-- ff_alt_text_enforcement: when true, hard-block save/publish; when false, soft-warn only.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS alt_text_enforcement_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_alt_text_enforcement BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.alt_text_enforcement_enabled IS
    'Plan 12.5: Enables alt-text prompts, AI suggestions, and accessibility coverage in course authoring.';
COMMENT ON COLUMN settings.platform_app_settings.ff_alt_text_enforcement IS
    'Plan 12.5: When true (and enforcement enabled), blocks content save until all images have alt text or are marked decorative.';
