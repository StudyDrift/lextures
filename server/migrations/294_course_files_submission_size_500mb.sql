-- Allow large submission attachments (e.g. video) imported from Canvas.
-- Aligns with course.submission_feedback_media byte_size cap (migration 102).

ALTER TABLE course.course_files DROP CONSTRAINT IF EXISTS course_files_byte_size_check;
ALTER TABLE course.course_files ADD CONSTRAINT course_files_byte_size_check
    CHECK (byte_size >= 0 AND byte_size <= 524288000);