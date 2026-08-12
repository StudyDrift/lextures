ALTER TABLE institution_inquiries DROP CONSTRAINT IF EXISTS institution_inquiries_first_touch_len;
ALTER TABLE institution_inquiries DROP COLUMN IF EXISTS crm_opportunity_id;
ALTER TABLE institution_inquiries DROP COLUMN IF EXISTS crm_lead_id;
ALTER TABLE institution_inquiries DROP COLUMN IF EXISTS first_touch_channel;
