-- AC.2 — Pre-assessment binding & adaptation profile columns/tables.

-- Extend per-learner adaptation decision (AC.1 skeleton).
ALTER TABLE course.adaptation_profiles
    ADD COLUMN IF NOT EXISTS target_bloom course.bloom_level,
    ADD COLUMN IF NOT EXISTS reading_level_pref TEXT,
    ADD COLUMN IF NOT EXISTS modality_pref TEXT,
    ADD COLUMN IF NOT EXISTS axis_set TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_neutral BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.adaptation_profiles.payload_json IS
    'Explainability snapshot: {conceptGaps:[{conceptId,gap}], misconceptions:[id], meanGap, priorRecord:bool}. No PII/free-text.';
COMMENT ON COLUMN course.adaptation_profiles.is_neutral IS
    'True when profile is a safe base fallback (missing mastery / compute error). Signature is base.';
COMMENT ON COLUMN course.adaptation_profiles.reading_level_pref IS
    'Reading-level preference (e.g. grade band or default).';
COMMENT ON COLUMN course.adaptation_profiles.modality_pref IS
    'Modality preference: text | worked_example | visual | default.';

-- Per-unit trigger config for when/how profiles are computed.
ALTER TABLE course.adaptive_content_units
    ADD COLUMN IF NOT EXISTS trigger_mode TEXT NOT NULL DEFAULT 'pre_quiz'
        CHECK (trigger_mode IN ('pre_quiz','diagnostic_first_visit','mastery_snapshot')),
    ADD COLUMN IF NOT EXISTS mastery_freshness_days SMALLINT NOT NULL DEFAULT 30
        CHECK (mastery_freshness_days >= 0);

COMMENT ON COLUMN course.adaptive_content_units.trigger_mode IS
    'How a learner profile is obtained: pre_quiz (default), diagnostic_first_visit, or mastery_snapshot.';
COMMENT ON COLUMN course.adaptive_content_units.mastery_freshness_days IS
    'For mastery_snapshot: only use learner_concept_states with last_seen_at within this many days.';

-- Explicit concepts a unit adapts around (in addition to those inferred from pre-check tags / outcomes).
CREATE TABLE IF NOT EXISTS course.adaptive_content_unit_concepts (
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    concept_id UUID NOT NULL REFERENCES course.concepts (id) ON DELETE CASCADE,
    PRIMARY KEY (unit_id, concept_id)
);

CREATE INDEX IF NOT EXISTS idx_ac_unit_concepts_concept
    ON course.adaptive_content_unit_concepts (concept_id);

COMMENT ON TABLE course.adaptive_content_unit_concepts IS
    'AC.2: Explicit concept set for an adaptive content unit.';
