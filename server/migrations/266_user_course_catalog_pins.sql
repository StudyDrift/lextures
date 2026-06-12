-- Per-user pinned courses for quick access from the sidebar (Zen-style shortcuts).

CREATE TABLE course.user_course_catalog_pins (
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, course_id)
);

CREATE INDEX idx_user_course_catalog_pins_user_sort
    ON course.user_course_catalog_pins (user_id, sort_order);

COMMENT ON TABLE course.user_course_catalog_pins IS
    'Pinned courses shown as shortcuts under the sidebar search bar.';