-- 438: Course grade levels as multi-value array (multi-select on create/settings)
-- Migrates legacy single grade_level TEXT → grade_levels TEXT[].
-- Idempotent: safe when re-applied after grade_level was already dropped (e.g. repair_345).

ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS grade_levels TEXT[];

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'course'
      AND table_name = 'courses'
      AND column_name = 'grade_level'
  ) THEN
    UPDATE course.courses
    SET grade_levels = ARRAY[grade_level]
    WHERE grade_level IS NOT NULL
      AND btrim(grade_level) <> ''
      AND (grade_levels IS NULL OR cardinality(grade_levels) = 0);
  END IF;
END $$;

DROP INDEX IF EXISTS courses_grade_level_idx;
ALTER TABLE course.courses DROP COLUMN IF EXISTS grade_level;

CREATE INDEX IF NOT EXISTS courses_grade_levels_gin_idx
  ON course.courses USING GIN (grade_levels);
