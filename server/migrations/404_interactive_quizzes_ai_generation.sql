-- IQ.10 — AI-assisted quiz kit generation (async jobs + question provenance).

CREATE TABLE IF NOT EXISTS quizgame.generation_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kit_id          UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE CASCADE,
    course_id       UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    requested_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    source_type     TEXT NOT NULL,
    source_ref      JSONB NOT NULL DEFAULT '{}'::jsonb,
    params          JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'queued',
    provider        TEXT,
    model           TEXT,
    usage_id        UUID,
    error           TEXT,
    result_summary  JSONB,
    progress        INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    CONSTRAINT quizgame_generation_jobs_source_type_chk
        CHECK (source_type IN ('topic', 'passage', 'course_content_ref')),
    CONSTRAINT quizgame_generation_jobs_status_chk
        CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'canceled')),
    CONSTRAINT quizgame_generation_jobs_progress_chk
        CHECK (progress BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_genjobs_kit
    ON quizgame.generation_jobs (kit_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_quizgame_genjobs_status
    ON quizgame.generation_jobs (status)
    WHERE status IN ('queued', 'running');

COMMENT ON TABLE quizgame.generation_jobs IS
    'IQ.10: Async AI quiz-kit generation jobs. source_ref never holds student data.';

ALTER TABLE quizgame.questions
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'authored',
    ADD COLUMN IF NOT EXISTS needs_review BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS generation_job_id UUID REFERENCES quizgame.generation_jobs (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS generation_confidence REAL;

DO $$ BEGIN
    ALTER TABLE quizgame.questions
        ADD CONSTRAINT quizgame_questions_source_chk
        CHECK (source IN ('authored', 'ai_generated', 'bank_import'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON COLUMN quizgame.questions.source IS
    'IQ.10: authored | ai_generated | bank_import provenance.';
COMMENT ON COLUMN quizgame.questions.needs_review IS
    'IQ.10: True when AI confidence is low or teacher has not accepted yet.';
COMMENT ON COLUMN quizgame.questions.generation_job_id IS
    'IQ.10: Job that produced this AI-drafted question.';
COMMENT ON COLUMN quizgame.questions.generation_confidence IS
    'IQ.10: Model confidence 0–1 when source=ai_generated; NULL otherwise.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_iq_ai_generation BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_iq_ai_generation IS
    'IQ.10: Platform sub-flag for AI-assisted live-quiz kit generation. Default OFF; requires ff_interactive_quizzes and configured AI.';

INSERT INTO settings.system_prompts (key, label, content)
VALUES (
    'live_quiz_kit_generation',
    'Live quiz kit AI generation',
    $PROMPT$You generate draft quiz questions for a live classroom game (Interactive Quizzes / Live Quizzes).
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object:
{"questions":[...],"suggestedSubject":"...","suggestedGradeBand":"..."}

Each question object uses camelCase keys matching this schema:
- questionType (string, required): one of exactly: mc_single, mc_multiple, true_false, type_answer, numeric, poll, ordering, word_cloud
- prompt (string, required): clear stem, age-appropriate, no HTML
- options (array of {id, text, isCorrect}): required for mc_single, mc_multiple, true_false, poll, ordering
  - mc_single: 2–6 options, exactly one isCorrect=true
  - mc_multiple: 2–6 options, at least one isCorrect=true
  - true_false: exactly [{id:"true",text:"True",isCorrect}, {id:"false",text:"False",isCorrect}] with one correct
  - poll / word_cloud: no correct answers required (isCorrect may be false for all); word_cloud may use options=[]
  - ordering: 3–6 options; correctAnswer.order lists option ids in correct sequence
- correctAnswer (object, when needed):
  - type_answer: {"accepted":[{"text":"...","matchMode":"case_insensitive"}]}
  - numeric: {"value": number, "tolerance": number, "unit": optional string}
  - ordering: {"order":["id1","id2",...]}
  - omit for poll and word_cloud
- timeLimitSeconds (integer, 5–240, default 20)
- explanation (string, optional): short rationale for the correct answer (teacher-facing)
- confidence (number 0–1): your confidence the item is well-formed and factually sound
- difficulty (string, optional): easy | medium | hard

Rules:
- Honour the requested count, allowed types, difficulty/grade band, and language.
- For passage or course-content sources, ground every question and correct answer in the source text; do not invent unsupported facts.
- For topic-only sources, prefer widely taught, non-controversial facts; mark lower confidence when uncertain.
- Distractors for multiple choice must be plausible for the grade band.
- Never include student personal data, PII, or unsafe content.
- Keep prompts projector-friendly (concise).$PROMPT$
)
ON CONFLICT (key) DO NOTHING;
