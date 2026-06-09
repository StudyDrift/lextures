-- Per-user course catalog UI: view mode, kanban labels, nicknames, and kanban placements.

CREATE TABLE course.user_course_catalog_prefs (
    user_id UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    view_type TEXT NOT NULL DEFAULT 'cards'
        CHECK (view_type IN ('cards', 'list', 'gallery', 'table', 'status')),
    kanban_column_labels JSONB NOT NULL DEFAULT '{
        "todo": "Todo",
        "in-progress": "In progress",
        "done": "Done",
        "hidden": "Hidden"
    }'::jsonb,
    hidden_column_expanded BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE course.user_course_catalog_nicknames (
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    nickname TEXT NOT NULL CHECK (char_length(btrim(nickname)) >= 1 AND char_length(nickname) <= 120),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, course_id)
);

CREATE TABLE course.user_course_kanban_placement (
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    column_id TEXT NOT NULL CHECK (column_id IN ('todo', 'in-progress', 'done', 'hidden')),
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, course_id)
);

CREATE INDEX idx_user_course_kanban_placement_user_column_sort
    ON course.user_course_kanban_placement (user_id, column_id, sort_order);

COMMENT ON TABLE course.user_course_catalog_prefs IS
    'Signed-in user preferences for the course catalog page (view mode and kanban labels).';
COMMENT ON TABLE course.user_course_catalog_nicknames IS
    'Optional per-user nicknames for enrolled courses in the catalog.';
COMMENT ON TABLE course.user_course_kanban_placement IS
    'Manual kanban column placement for a user''s course catalog board.';
