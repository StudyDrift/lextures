-- AC.3 — Adaptive content generation engine: variant metadata, content versioning, key terms, system prompt.

-- Generation metadata on stored variants.
ALTER TABLE course.content_variants
    ADD COLUMN IF NOT EXISTS prompt_version TEXT NOT NULL DEFAULT 'v1',
    ADD COLUMN IF NOT EXISTS content_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS completion_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS a11y_flags JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN course.content_variants.prompt_version IS
    'AC.3: Version stamp of the adaptive_content_variant system prompt used to produce this row.';
COMMENT ON COLUMN course.content_variants.content_version IS
    'AC.3: adaptive_content_units.content_version at generation time; mismatch ⇒ supersede / regenerate.';
COMMENT ON COLUMN course.content_variants.prompt_tokens IS
    'AC.3: Prompt token count from the generation call (also logged to analytics.ai_usage_log).';
COMMENT ON COLUMN course.content_variants.completion_tokens IS
    'AC.3: Completion token count from the generation call.';
COMMENT ON COLUMN course.content_variants.a11y_flags IS
    'AC.3: Post-gen accessibility lint findings (JSON array of string codes).';

-- Monotonic version bumped whenever the base content page changes, to invalidate variants.
ALTER TABLE course.adaptive_content_units
    ADD COLUMN IF NOT EXISTS content_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS min_fidelity REAL NOT NULL DEFAULT 0.85
        CHECK (min_fidelity IS NULL OR (min_fidelity BETWEEN 0 AND 1));

-- Tighten: column is NOT NULL with default; re-apply CHECK cleanly if needed.
ALTER TABLE course.adaptive_content_units
    DROP CONSTRAINT IF EXISTS adaptive_content_units_min_fidelity_check;
ALTER TABLE course.adaptive_content_units
    ADD CONSTRAINT adaptive_content_units_min_fidelity_check
        CHECK (min_fidelity BETWEEN 0 AND 1);

COMMENT ON COLUMN course.adaptive_content_units.content_version IS
    'AC.3: Bumped when base content changes so cached variants are superseded.';
COMMENT ON COLUMN course.adaptive_content_units.min_fidelity IS
    'AC.3: Minimum fidelity_score (0–1) required for a variant to pass the fidelity gate (default 0.85).';

-- Instructor-marked terms/facts that MUST survive any rewrite (glossary/standards anchors).
CREATE TABLE IF NOT EXISTS course.adaptive_content_key_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    term TEXT NOT NULL,
    must_appear BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ac_key_terms_unit ON course.adaptive_content_key_terms (unit_id);

COMMENT ON TABLE course.adaptive_content_key_terms IS
    'AC.3: Instructor-marked key terms that hard-fail fidelity when missing from a generated variant.';

-- Versioned generation prompt (mirrors adaptive_quiz seeding).
INSERT INTO settings.system_prompts (key, label, content)
VALUES (
    'adaptive_content_variant',
    'Adaptive content variant (per-learner rewrite)',
    $PROMPT$You adapt an existing LMS content page for one learner. You are given the AUTHORITATIVE base content, an emphasis mode, the learner's concept gaps and misconceptions (by name), a target cognitive level, and allowed adaptation axes.

Rewrite or re-emphasize the base to fit the learner. Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"sections":[{"heading":"...","markdown":"..."}]}.

Absolute rules:
- Preserve every fact, definition, standard reference, formula, code block, and listed key term from the base exactly.
- Introduce no claim, statistic, or example not entailed by the base content.
- Never remove required terminology.
- heading: short section title without markdown # prefixes; use "" for a lead-in block with no heading.
- markdown: body content in Markdown only (paragraphs, lists, emphasis, links, fenced code, LaTeX). Do NOT put ## headings inside markdown — use separate section objects instead.
- Prefer 2–12 clear sections; return between 1 and 20 sections.
- Write in the same language as the base content; preserve RTL directionality if present.
- Suggested images must include alt-text placeholders in markdown (e.g. ![description](url)).

Per-mode guidance (from emphasisMode):
- introduce: build from prerequisites; define terms carefully; scaffold foundational understanding.
- reinforce: keep core structure; add one worked example or practice cue on gap concepts.
- compress: condense aggressively for near-masters; skip proven material; keep all key terms and formulas.
- remediate: explicitly name each listed misconception and correct it with a clear contrast to the accurate concept.

If the base content is empty or unusable, return {"sections":[]}.$PROMPT$
)
ON CONFLICT (key) DO NOTHING;
