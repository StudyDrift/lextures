-- Student notebooks (markdown pages + groups), synced from web and mobile editors.
-- One row per learner per course; course_code '__lextures_global__' is the learner-wide notebook.

CREATE TABLE analytics.student_notebooks (
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_code TEXT NOT NULL,
    data        JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, course_code)
);

COMMENT ON TABLE analytics.student_notebooks IS
    'Learner notebook documents (format v2: pages/groups with markdown content); last-write-wins sync across web and mobile clients.';
