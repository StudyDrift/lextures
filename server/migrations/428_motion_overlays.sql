-- AN.5: kill-switch for overlay / surface motion (dialogs, sheets, menus, toasts, tooltips).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_motion_overlays BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_motion_overlays IS
    'AN.5: Overlay/surface motion (modals, sheets, drawers, menus, toasts, tooltips). Default ON; set false to disable instantly. Collapsed into ff_motion_navigation.';
