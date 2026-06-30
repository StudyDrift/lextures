-- Plan 19.2 — Auto-generated lesson plans, assessments, and differentiation.

CREATE TABLE IF NOT EXISTS jobs.lesson_generation_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instructor_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id       UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    input_params    JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    result            JSONB,
    error_message     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS lesson_generation_jobs_status_created_idx
    ON jobs.lesson_generation_jobs (status, created_at)
    WHERE status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS lesson_generation_jobs_instructor_created_idx
    ON jobs.lesson_generation_jobs (instructor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS lesson_generation_jobs_course_created_idx
    ON jobs.lesson_generation_jobs (course_id, created_at DESC);

ALTER TABLE course.course_structure_items
    ADD COLUMN IF NOT EXISTS provenance JSONB;

COMMENT ON COLUMN course.course_structure_items.provenance IS
    'AI content provenance metadata: generated_by, model_id, generation_ts (plan 19.9 / 19.2).';

ALTER TABLE course.module_quizzes
    ADD COLUMN IF NOT EXISTS provenance JSONB;

COMMENT ON COLUMN course.module_quizzes.provenance IS
    'AI content provenance metadata for generated quiz content (plan 19.9 / 19.2).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_lesson_generator BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_lesson_generator IS
    'Enables the AI lesson generator wizard for instructors (plan 19.2).';

INSERT INTO settings.system_prompts (key, label, content)
VALUES (
    'lesson_generation_plan',
    'Lesson plan outline generation',
    $PROMPT$You are an expert K-12 and higher-education curriculum designer. Given a learning objective, grade level, and subject, write a complete lesson plan in Markdown.

Include these sections when appropriate:
- Warm-up / hook
- Learning objective (restated for students)
- Direct instruction outline
- Guided practice
- Independent or group activity
- Closure / exit ticket preview
- Materials needed
- Timing notes when duration is provided

Write in a professional, accessible tone for the specified grade level. Output only Markdown — no preamble, no code fences, no commentary.$PROMPT$
)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings.system_prompts (key, label, content)
VALUES (
    'lesson_generation_activity',
    'Differentiated activity generation',
    $PROMPT$You are an expert curriculum designer creating a differentiated classroom activity. Given a learning objective, grade level, subject, and differentiation level, write a single activity in Markdown.

Differentiation levels:
- below_grade: simpler vocabulary, more scaffolding, reduced task complexity
- on_grade: grade-appropriate vocabulary and task demand
- advanced: higher vocabulary complexity, extension or enrichment task demand
- ell: supports for English language learners (visuals cues, sentence frames, cognates)
- iep: accessible task design with clear steps and optional accommodations

Output only Markdown describing the activity (title, instructions, materials, grouping). No preamble or code fences.$PROMPT$
)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings.system_prompts (key, label, content)
VALUES (
    'lesson_generation_rubric',
    'Open-ended task rubric generation',
    $PROMPT$You generate assignment rubrics for an LMS. Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must match this schema:
{
  "title": "optional string",
  "criteria": [
    {
      "id": "uuid string",
      "title": "criterion title",
      "description": "optional string",
      "levels": [
        { "label": "level name", "points": number, "description": "optional string" }
      ]
    }
  ]
}

Rules:
- Include 3–5 criteria aligned to the learning objective.
- Each criterion has 3–4 levels with ascending points.
- Use new random UUIDs for each criterion id.
- Keep language appropriate for the grade level.$PROMPT$
)
ON CONFLICT (key) DO NOTHING;
