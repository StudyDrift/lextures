-- Companion to: 438_course_grade_levels_array.sql

ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS grade_level TEXT;

UPDATE course.courses
SET grade_level = grade_levels[1]
WHERE grade_levels IS NOT NULL
  AND cardinality(grade_levels) > 0
  AND (grade_level IS NULL OR btrim(grade_level) = '');

DROP INDEX IF EXISTS courses_grade_levels_gin_idx;
ALTER TABLE course.courses DROP COLUMN IF EXISTS grade_levels;

CREATE INDEX IF NOT EXISTS courses_grade_level_idx
  ON course.courses (grade_level)
  WHERE grade_level IS NOT NULL;
