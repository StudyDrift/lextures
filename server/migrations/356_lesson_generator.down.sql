DROP TABLE IF EXISTS jobs.lesson_generation_jobs;

ALTER TABLE course.course_structure_items DROP COLUMN IF EXISTS provenance;
ALTER TABLE course.module_quizzes DROP COLUMN IF EXISTS provenance;
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS ff_lesson_generator;

DELETE FROM settings.system_prompts
WHERE key IN ('lesson_generation_plan', 'lesson_generation_activity', 'lesson_generation_rubric');
