-- Course Files feature: folder hierarchy + file items + per-course feature flag.
-- Separate from course.course_files (embedded content images); this is the Drive-like file space.

ALTER TABLE course.courses ADD COLUMN files_enabled BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE course.file_folders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id  UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES course.file_folders (id) ON DELETE CASCADE,
    name       TEXT NOT NULL CHECK (char_length(name) > 0 AND char_length(name) <= 255),
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_file_folders_course ON course.file_folders (course_id);
CREATE INDEX idx_file_folders_parent ON course.file_folders (parent_id);

CREATE TABLE course.file_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    folder_id         UUID REFERENCES course.file_folders (id) ON DELETE SET NULL,
    storage_key       TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    display_name      TEXT NOT NULL,
    mime_type         TEXT NOT NULL DEFAULT 'application/octet-stream',
    byte_size         BIGINT NOT NULL CHECK (byte_size >= 0),
    uploaded_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    canvas_file_id    BIGINT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_file_items_storage_key ON course.file_items (storage_key);
CREATE INDEX idx_file_items_course_folder ON course.file_items (course_id, folder_id);
CREATE INDEX idx_file_items_canvas ON course.file_items (course_id, canvas_file_id) WHERE canvas_file_id IS NOT NULL;
