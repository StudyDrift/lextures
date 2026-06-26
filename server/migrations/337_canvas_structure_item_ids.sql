-- Persist the Canvas identifiers for imported assignment/quiz items so grade sync can target the
-- correct Canvas assignment by id instead of matching on title (titles are not unique).
-- canvas_assignment_id is the Canvas assignment that grades are pushed to: for assignment items it
-- is the assignment id; for quiz items it is the quiz's backing assignment id.
ALTER TABLE course.course_structure_items
  ADD COLUMN IF NOT EXISTS canvas_assignment_id BIGINT,
  ADD COLUMN IF NOT EXISTS canvas_quiz_id BIGINT;

-- Speeds up reverse lookups (course + Canvas assignment id) used during grade sync / re-import.
CREATE INDEX IF NOT EXISTS idx_course_structure_items_canvas_assignment
  ON course.course_structure_items (course_id, canvas_assignment_id)
  WHERE canvas_assignment_id IS NOT NULL;
