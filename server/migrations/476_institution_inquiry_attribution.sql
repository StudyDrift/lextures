ALTER TABLE institution_inquiries
    ADD COLUMN IF NOT EXISTS first_touch_channel TEXT NOT NULL DEFAULT 'direct',
    ADD COLUMN IF NOT EXISTS crm_lead_id TEXT,
    ADD COLUMN IF NOT EXISTS crm_opportunity_id TEXT;

ALTER TABLE institution_inquiries
    ADD CONSTRAINT institution_inquiries_first_touch_len
    CHECK (char_length(first_touch_channel) BETWEEN 1 AND 80);

COMMENT ON COLUMN institution_inquiries.first_touch_channel IS
    'Privacy-safe first-touch acquisition channel passed to CRM lead/opportunity exports.';
