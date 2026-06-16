-- Learning paths / course bundles (plan 15.4).

CREATE SCHEMA IF NOT EXISTS learningpath;

CREATE TABLE learningpath.learning_paths (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    slug               TEXT UNIQUE,
    bundle_price_cents INT,
    stripe_product_id  TEXT,
    is_public          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_learning_paths_creator ON learningpath.learning_paths (creator_id, updated_at DESC);
CREATE INDEX idx_learning_paths_public ON learningpath.learning_paths (is_public, created_at DESC)
    WHERE is_public = TRUE;

COMMENT ON TABLE learningpath.learning_paths IS
    'Ordered multi-course bundles / specializations for self-learners (plan 15.4).';

CREATE TABLE learningpath.learning_path_courses (
    path_id    UUID NOT NULL REFERENCES learningpath.learning_paths (id) ON DELETE CASCADE,
    course_id  UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    position   INT NOT NULL CHECK (position >= 0),
    PRIMARY KEY (path_id, course_id)
);

CREATE UNIQUE INDEX idx_learning_path_courses_position
    ON learningpath.learning_path_courses (path_id, position);

CREATE INDEX idx_learning_path_courses_course ON learningpath.learning_path_courses (course_id);

COMMENT ON TABLE learningpath.learning_path_courses IS
    'Ordered constituent courses within a learning path (plan 15.4).';

CREATE TABLE learningpath.path_enrollments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    path_id     UUID NOT NULL REFERENCES learningpath.learning_paths (id) ON DELETE CASCADE,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE (user_id, path_id)
);

CREATE INDEX idx_path_enrollments_user ON learningpath.path_enrollments (user_id, enrolled_at DESC);
CREATE INDEX idx_path_enrollments_path ON learningpath.path_enrollments (path_id);

COMMENT ON TABLE learningpath.path_enrollments IS
    'Learner enrollment in a learning path; bulk course enrollments are created atomically (plan 15.4).';

-- Minimal entitlement store for path bundle purchases (extended by plan 15.3 Stripe billing).
CREATE SCHEMA IF NOT EXISTS billing;

CREATE TABLE billing.user_entitlements (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    entitlement_type  TEXT NOT NULL,
    path_id           UUID REFERENCES learningpath.learning_paths (id) ON DELETE CASCADE,
    course_id         UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    stripe_event_id   TEXT UNIQUE,
    amount_paid_cents INT NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT 'usd',
    valid_from        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until       TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'expired', 'refunded')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (path_id IS NOT NULL OR course_id IS NOT NULL)
);

CREATE INDEX idx_entitlements_user_path
    ON billing.user_entitlements (user_id, path_id, status)
    WHERE path_id IS NOT NULL;

CREATE INDEX idx_entitlements_user_course
    ON billing.user_entitlements (user_id, course_id, status)
    WHERE course_id IS NOT NULL;

COMMENT ON TABLE billing.user_entitlements IS
    'Purchase and subscription entitlements for paid paths and courses (plans 15.3, 15.4).';

-- Optional per-course list price for bundle savings display (full billing in plan 15.3).
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS list_price_cents INT;

COMMENT ON COLUMN course.courses.list_price_cents IS
    'Optional self-learner list price in cents; used for path bundle savings callouts (plans 15.3, 15.4).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_learning_paths BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_learning_paths IS
    'Enables learning paths / course bundles for self-learners (plan 15.4). Managed in Settings → Global platform.';
