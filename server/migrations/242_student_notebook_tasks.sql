-- Student notebook tasks (checkbox items from /task or /todo slash commands).

CREATE TABLE analytics.student_notebook_tasks (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_code      TEXT NOT NULL,
    notebook_page_id TEXT NOT NULL,
    task_text        TEXT NOT NULL DEFAULT '',
    completed        BOOLEAN NOT NULL DEFAULT false,
    due_at           TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_student_notebook_tasks_user_open
    ON analytics.student_notebook_tasks (user_id, completed, due_at NULLS LAST, created_at DESC);

COMMENT ON TABLE analytics.student_notebook_tasks IS
    'Learner tasks created in course or global notebooks; synced from client-side notebook editor.';
