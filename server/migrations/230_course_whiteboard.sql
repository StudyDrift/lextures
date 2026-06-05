ALTER TABLE course.courses ADD COLUMN whiteboard_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS course.whiteboards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    canvas_data JSONB NOT NULL DEFAULT '[]',
    created_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_whiteboards_course ON course.whiteboards (course_id);
