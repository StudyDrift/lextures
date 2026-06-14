-- ff_ui_mode has had a Row/Write field and merge logic in platformconfig for some time, but no
-- prior migration ever created the column (the repo never SELECTed it; it was effectively
-- env-only). Add it now so the platform toggle persists and resolves. Idempotent.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_ui_mode BOOLEAN;
