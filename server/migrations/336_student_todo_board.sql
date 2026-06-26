-- Per-learner kanban placement for the global Todos page (weekday columns + done).

CREATE TABLE analytics.student_todo_board_placement (
    user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    item_key   TEXT NOT NULL,
    column_id  TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_key)
);

CREATE INDEX idx_student_todo_board_placement_user_column
    ON analytics.student_todo_board_placement (user_id, column_id, sort_order);

COMMENT ON TABLE analytics.student_todo_board_placement IS
    'Learner-specific column and sort order for items on the global Todos kanban board.';