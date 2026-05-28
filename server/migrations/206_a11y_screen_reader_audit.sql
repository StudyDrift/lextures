-- Plan 12.1: Screen-reader audit phase 1 — ARIA remediation for TipTap editor, gradebook grid,
-- dnd-kit module reorder, and command palette.
-- Feature flag defaults false until QA sign-off; flip to true to activate in prod.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_a11y_aria_audit_phase1 BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.ff_a11y_aria_audit_phase1 IS
    'Plan 12.1: Enables ARIA remediation changes across the block editor, gradebook grid, dnd-kit module reorder, and command palette. Default false; flip true after QA screen-reader sign-off.';
