-- Per-user row layout for pinned course shortcuts (max 4 per row in the sidebar).

ALTER TABLE course.user_course_catalog_pins
    ADD COLUMN row_index INT NOT NULL DEFAULT 0 CHECK (row_index >= 0);

-- Backfill existing flat sort_order into rows of up to four pins.
UPDATE course.user_course_catalog_pins
SET
    row_index = sort_order / 4,
    sort_order = sort_order % 4;

DROP INDEX IF EXISTS course.idx_user_course_catalog_pins_user_sort;

CREATE INDEX idx_user_course_catalog_pins_user_row_sort
    ON course.user_course_catalog_pins (user_id, row_index, sort_order);

COMMENT ON COLUMN course.user_course_catalog_pins.row_index IS
    'Zero-based row in the sidebar pin grid; sort_order is the position within the row (0-3).';