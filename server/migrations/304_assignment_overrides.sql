-- 2.15 Differentiated Assignments: unified assign-to targeting (everyone/section/group/student)
-- with per-target due/availability overrides. Generalizes course.section_assignment_overrides
-- (migration 131) into one model used by every effective-date consumer.

CREATE TABLE course.assignment_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    structure_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    item_type TEXT NOT NULL CHECK (item_type IN ('assignment', 'quiz')),
    target_type TEXT NOT NULL CHECK (target_type IN ('everyone', 'section', 'group', 'student')),
    target_id UUID NULL,
    due_at TIMESTAMPTZ NULL,
    available_from TIMESTAMPTZ NULL,
    available_until TIMESTAMPTZ NULL,
    created_by UUID NULL REFERENCES "user".users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assignment_overrides_target_id_required CHECK (
        (target_type = 'everyone' AND target_id IS NULL) OR
        (target_type <> 'everyone' AND target_id IS NOT NULL)
    )
);

-- At most one 'everyone' target per item.
CREATE UNIQUE INDEX idx_assignment_overrides_unique_everyone
    ON course.assignment_overrides (structure_item_id)
    WHERE target_type = 'everyone';

-- At most one row per (item, specific target).
CREATE UNIQUE INDEX idx_assignment_overrides_unique_targeted
    ON course.assignment_overrides (structure_item_id, target_type, target_id)
    WHERE target_type <> 'everyone';

CREATE INDEX idx_assignment_overrides_item ON course.assignment_overrides (structure_item_id);
CREATE INDEX idx_assignment_overrides_target ON course.assignment_overrides (target_type, target_id);

-- Backfill: existing section overrides become 'section' targets in the unified table.
INSERT INTO course.assignment_overrides (structure_item_id, item_type, target_type, target_id, due_at, available_from, available_until, created_at)
SELECT sao.structure_item_id, csi.kind, 'section', sao.section_id, sao.due_at, sao.available_from, sao.available_until, NOW()
FROM course.section_assignment_overrides sao
INNER JOIN course.course_structure_items csi ON csi.id = sao.structure_item_id
WHERE csi.kind IN ('assignment', 'quiz')
ON CONFLICT DO NOTHING;
