-- Issue #775 — kid-safe messaging: teachers a parent approved as message contacts for a managed learner.
-- Parents (active parent_student_links) are always allowed; this table only lists extra approved contacts.

CREATE TABLE IF NOT EXISTS communication.managed_learner_contacts (
    student_user_id  UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    contact_user_id  UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    approved_by      UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (student_user_id, contact_user_id),
    CHECK (student_user_id <> contact_user_id)
);

CREATE INDEX IF NOT EXISTS managed_learner_contacts_contact
    ON communication.managed_learner_contacts (contact_user_id);

COMMENT ON TABLE communication.managed_learner_contacts IS
    'Issue #775: course teachers a parent approved to exchange inbox messages with a managed learner.';
