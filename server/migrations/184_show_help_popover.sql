-- Migration 184: Add show_help_popover user preference
ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS show_help_popover BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN "user".users.show_help_popover IS 'Persisted LMS UI setting to show or hide the help popover in the bottom right corner of all pages. Default true.';
