-- Pending student enrollment invitations (approve/decline via inbox + courses catalog).

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS invitation_pending BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_course_enrollments_invitation_pending
    ON course.course_enrollments (user_id, course_id)
    WHERE invitation_pending;

ALTER TABLE communication.messages
    ADD COLUMN IF NOT EXISTS metadata JSONB;