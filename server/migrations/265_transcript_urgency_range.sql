-- Support business-day ranges for mail transcript urgency (e.g. 3–5 standard, 1–2 rush).

ALTER TABLE transcripts.transcript_requests
    ADD COLUMN IF NOT EXISTS urgency_days_min INT
        CHECK (
            urgency_days_min IS NULL
            OR (urgency_days_min > 0 AND urgency_days_min <= urgency_days)
        );

COMMENT ON COLUMN transcripts.transcript_requests.urgency_days_min IS
    'Lower bound when urgency is a range (mail standard/rush). Null for single-day values.';