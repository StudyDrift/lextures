-- IQ.11 follow-up: default guest-join policy to teacher_mediated so existing
-- FFIqGuestJoin + host allowGuests behaviour is unchanged until admins tighten it.

ALTER TABLE settings.platform_app_settings
    ALTER COLUMN iq_guest_join_policy SET DEFAULT 'teacher_mediated';

UPDATE settings.platform_app_settings
SET iq_guest_join_policy = 'teacher_mediated'
WHERE iq_guest_join_policy = 'disabled';

COMMENT ON COLUMN settings.platform_app_settings.iq_guest_join_policy IS
    'IQ.11: disabled | teacher_mediated | open — platform default guest-join policy. Default teacher_mediated preserves FF + host allowGuests behaviour.';
