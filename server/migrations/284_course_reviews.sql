-- Course reviews & ratings for self-learner catalog trust signals (plan 15.7).

CREATE TABLE course.course_reviews (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id        UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    reviewer_id      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    rating           SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review_text      TEXT,
    creator_response TEXT,
    is_flagged       BOOLEAN NOT NULL DEFAULT FALSE,
    is_removed       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, reviewer_id)
);

COMMENT ON TABLE course.course_reviews IS
    'Learner star ratings and optional text reviews for self-paced courses (plan 15.7).';

CREATE INDEX idx_course_reviews_course
    ON course.course_reviews (course_id, created_at DESC)
    WHERE is_removed = FALSE;

CREATE INDEX idx_course_reviews_flagged
    ON course.course_reviews (is_flagged, created_at DESC)
    WHERE is_flagged = TRUE AND is_removed = FALSE;

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS rating_sum   INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rating_count INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN course.courses.rating_sum IS
    'Running sum of active (non-removed) review ratings for O(1) average updates (plan 15.7).';
COMMENT ON COLUMN course.courses.rating_count IS
    'Count of active (non-removed) reviews; average_rating = rating_sum / rating_count (plan 15.7).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_course_reviews BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_course_reviews IS
    'Enables course star ratings and learner reviews on catalog and course pages (plan 15.7).';
