UPDATE settings.platform_app_settings
SET iq_guest_join_policy = 'disabled'
WHERE iq_guest_join_policy = 'teacher_mediated';

ALTER TABLE settings.platform_app_settings
    ALTER COLUMN iq_guest_join_policy SET DEFAULT 'disabled';

COMMENT ON COLUMN settings.platform_app_settings.iq_guest_join_policy IS
    'IQ.11: disabled | teacher_mediated | open — platform default guest-join policy.';
