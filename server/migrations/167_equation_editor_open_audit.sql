-- 8.11 follow-up: allow equation_editor_open audit events.
-- Early applies of migration 164 only added equation_inserted; this brings constraints in line
-- with server/internal/httpserver/course_context.go.

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_event_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_event_kind_check CHECK (
    event_kind IN (
        'course_visit',
        'content_open',
        'content_leave',
        'equation_inserted',
        'equation_editor_open'
    )
);

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind;
ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_structure_item_kind_check CHECK (
    (event_kind = 'course_visit' AND structure_item_id IS NULL)
    OR (event_kind IN ('content_open', 'content_leave') AND structure_item_id IS NOT NULL)
    OR (event_kind IN ('equation_inserted', 'equation_editor_open'))
);
