-- 8.11: Math equation editor audit events in user.user_audit

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

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_structure_item_kind_check CHECK (
    (event_kind = 'course_visit' AND structure_item_id IS NULL)
    OR (event_kind IN ('content_open', 'content_leave') AND structure_item_id IS NOT NULL)
    OR (event_kind IN ('equation_inserted', 'equation_editor_open'))
);
