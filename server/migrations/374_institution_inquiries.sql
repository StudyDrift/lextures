-- Marketing institution "Request information" leads from lextures.com.
-- Email notification can be wired later; for now this is write-only storage.

CREATE TABLE IF NOT EXISTS institution_inquiries (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    organization_type    TEXT NOT NULL,
    organization_name    TEXT NOT NULL,
    contact_name         TEXT NOT NULL,
    email                TEXT NOT NULL,
    role                 TEXT,
    enrollment_size      TEXT NOT NULL,
    hosting_preference   TEXT NOT NULL,
    message              TEXT NOT NULL,
    ip_address           TEXT,
    user_agent           TEXT,
    status               TEXT NOT NULL DEFAULT 'new',
    CONSTRAINT institution_inquiries_org_type_len
        CHECK (char_length(organization_type) BETWEEN 1 AND 80),
    CONSTRAINT institution_inquiries_org_name_len
        CHECK (char_length(organization_name) BETWEEN 1 AND 200),
    CONSTRAINT institution_inquiries_contact_name_len
        CHECK (char_length(contact_name) BETWEEN 1 AND 200),
    CONSTRAINT institution_inquiries_email_len
        CHECK (char_length(email) BETWEEN 3 AND 320),
    CONSTRAINT institution_inquiries_role_len
        CHECK (role IS NULL OR char_length(role) <= 200),
    CONSTRAINT institution_inquiries_enrollment_len
        CHECK (char_length(enrollment_size) BETWEEN 1 AND 80),
    CONSTRAINT institution_inquiries_hosting_len
        CHECK (char_length(hosting_preference) BETWEEN 1 AND 120),
    CONSTRAINT institution_inquiries_message_len
        CHECK (char_length(message) BETWEEN 1 AND 5000),
    CONSTRAINT institution_inquiries_status_check
        CHECK (status IN ('new', 'contacted', 'closed', 'spam'))
);

COMMENT ON TABLE institution_inquiries IS
    'Institutional sales/leads from the marketing request-information form. Email delivery may be added later.';

CREATE INDEX IF NOT EXISTS institution_inquiries_created_at_idx
    ON institution_inquiries (created_at DESC);

CREATE INDEX IF NOT EXISTS institution_inquiries_status_created_idx
    ON institution_inquiries (status, created_at DESC);

CREATE INDEX IF NOT EXISTS institution_inquiries_email_idx
    ON institution_inquiries (lower(email));
