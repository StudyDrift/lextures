-- Transcript request delivery options and pickup instructions.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS pickup_instructions TEXT;

COMMENT ON COLUMN settings.transcripts_config.pickup_instructions IS
    'Instructions shown to students who choose in-person pickup (location, hours, ID requirements).';

ALTER TABLE transcripts.transcript_requests
    ADD COLUMN IF NOT EXISTS delivery_type TEXT NOT NULL DEFAULT 'email'
        CHECK (delivery_type IN ('email', 'mail', 'pickup')),
    ADD COLUMN IF NOT EXISTS delivery_email TEXT,
    ADD COLUMN IF NOT EXISTS delivery_address TEXT,
    ADD COLUMN IF NOT EXISTS urgency_days INT NOT NULL DEFAULT 1
        CHECK (urgency_days > 0 AND urgency_days <= 30),
    ADD COLUMN IF NOT EXISTS urgency_unit TEXT NOT NULL DEFAULT 'days'
        CHECK (urgency_unit IN ('days', 'business_days'));

COMMENT ON COLUMN transcripts.transcript_requests.delivery_type IS
    'How the student wants the transcript delivered: email, mail, or pickup.';
COMMENT ON COLUMN transcripts.transcript_requests.urgency_unit IS
    'Whether urgency_days counts calendar days (email) or business days (mail, pickup).';