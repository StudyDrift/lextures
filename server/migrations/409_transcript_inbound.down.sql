-- Companion to: 409_transcript_inbound.sql

DROP TABLE IF EXISTS transcripts.inbound_events;
DROP TABLE IF EXISTS transcripts.inbound_documents;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_transcript_inbound;
