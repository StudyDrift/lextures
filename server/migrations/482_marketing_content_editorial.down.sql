DROP TABLE IF EXISTS marketing.content_notification_log;
DROP TABLE IF EXISTS marketing.content_editorial_settings;
DROP TABLE IF EXISTS marketing.content_health_snapshots;
DROP TABLE IF EXISTS marketing.content_overrides;
DROP TABLE IF EXISTS marketing.content_link_health;
DROP TABLE IF EXISTS marketing.content_briefs;
DROP TABLE IF EXISTS marketing.content_reviews;
ALTER TABLE marketing.content_articles DROP COLUMN IF EXISTS review_submitted_at, DROP COLUMN IF EXISTS reviewer_id;
