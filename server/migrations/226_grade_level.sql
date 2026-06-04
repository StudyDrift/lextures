-- 226: Grade-level metadata for K-12 (plan 13.6)
-- Valid values: K, 1-12, K-2, 3-5, 6-8, 9-12, K-12 (NULL = unset / non-K12)
ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS grade_level TEXT;
ALTER TABLE "user".users   ADD COLUMN IF NOT EXISTS grade_level TEXT;
CREATE INDEX IF NOT EXISTS courses_grade_level_idx ON course.courses(grade_level) WHERE grade_level IS NOT NULL;
CREATE INDEX IF NOT EXISTS users_grade_level_idx ON "user".users(grade_level) WHERE grade_level IS NOT NULL;
