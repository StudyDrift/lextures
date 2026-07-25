-- 438: Course grade levels as multi-value array (multi-select on create/settings)
-- Migrates legacy single grade_level TEXT → grade_levels TEXT[].

ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS grade_levels TEXT[];

UPDATE course.courses
SET grade_levels = ARRAY[grade_level]
WHERE grade_level IS NOT NULL
  AND btrim(grade_level) <> ''
  AND (grade_levels IS NULL OR cardinality(grade_levels) = 0);

DROP INDEX IF EXISTS courses_grade_level_idx;
ALTER TABLE course.courses DROP COLUMN IF EXISTS grade_level;

CREATE INDEX IF NOT EXISTS courses_grade_levels_gin_idx
  ON course.courses USING GIN (grade_levels);
