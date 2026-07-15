-- Companion to: 374_institution_inquiries.sql

DROP INDEX IF EXISTS institution_inquiries_email_idx;
DROP INDEX IF EXISTS institution_inquiries_status_created_idx;
DROP INDEX IF EXISTS institution_inquiries_created_at_idx;
DROP TABLE IF EXISTS institution_inquiries;
