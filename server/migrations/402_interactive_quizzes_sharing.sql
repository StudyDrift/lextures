-- IQ.8 — Content library, templates, sharing & discovery.

-- Allow system-scoped templates without a course.
ALTER TABLE quizgame.kits
    ALTER COLUMN course_id DROP NOT NULL;

ALTER TABLE quizgame.kits
    ADD COLUMN IF NOT EXISTS is_template         BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS template_scope      TEXT,
    ADD COLUMN IF NOT EXISTS derived_from_kit_id UUID REFERENCES quizgame.kits (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS attribution         TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS subject             TEXT,
    ADD COLUMN IF NOT EXISTS grade_band          TEXT,
    ADD COLUMN IF NOT EXISTS language            TEXT,
    ADD COLUMN IF NOT EXISTS catalog_status      TEXT NOT NULL DEFAULT 'unlisted';

DO $$ BEGIN
    ALTER TABLE quizgame.kits
        ADD CONSTRAINT quizgame_kits_template_scope_chk
        CHECK (
            template_scope IS NULL
            OR template_scope IN ('system', 'org', 'course')
        );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE quizgame.kits
        ADD CONSTRAINT quizgame_kits_catalog_status_chk
        CHECK (catalog_status IN ('unlisted', 'pending', 'listed', 'rejected'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE quizgame.kits
        ADD CONSTRAINT quizgame_kits_course_or_system_template_chk
        CHECK (
            course_id IS NOT NULL
            OR (is_template = TRUE AND template_scope = 'system')
        );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS quizgame.kit_shares (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kit_id       UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE CASCADE,
    grantee_type TEXT NOT NULL,
    grantee_id   UUID,
    permission   TEXT NOT NULL DEFAULT 'copy',
    created_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quizgame_kit_shares_grantee_type_chk
        CHECK (grantee_type IN ('user', 'course', 'org_unit', 'org')),
    CONSTRAINT quizgame_kit_shares_permission_chk
        CHECK (permission IN ('view', 'copy', 'edit')),
    CONSTRAINT quizgame_kit_shares_grantee_id_chk
        CHECK (
            (grantee_type = 'org' AND grantee_id IS NULL)
            OR (grantee_type <> 'org' AND grantee_id IS NOT NULL)
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_quizgame_kit_shares_unique
    ON quizgame.kit_shares (kit_id, grantee_type, grantee_id, permission)
    NULLS NOT DISTINCT;

CREATE INDEX IF NOT EXISTS idx_quizgame_kit_shares_kit ON quizgame.kit_shares (kit_id);
CREATE INDEX IF NOT EXISTS idx_quizgame_kit_shares_grantee
    ON quizgame.kit_shares (grantee_type, grantee_id);

CREATE INDEX IF NOT EXISTS idx_quizgame_kits_catalog
    ON quizgame.kits (catalog_status)
    WHERE catalog_status = 'listed';

CREATE INDEX IF NOT EXISTS idx_quizgame_kits_templates
    ON quizgame.kits (is_template, template_scope)
    WHERE is_template = TRUE;

CREATE INDEX IF NOT EXISTS idx_quizgame_kits_title_fts
    ON quizgame.kits
    USING gin (to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')));

CREATE INDEX IF NOT EXISTS idx_quizgame_kits_subject ON quizgame.kits (subject)
    WHERE subject IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_quizgame_kits_grade_band ON quizgame.kits (grade_band)
    WHERE grade_band IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_quizgame_kits_language ON quizgame.kits (language)
    WHERE language IS NOT NULL;

COMMENT ON TABLE quizgame.kit_shares IS
    'IQ.8: Grants view/copy/edit access to a kit for a user, course, org unit, or org-wide.';
COMMENT ON COLUMN quizgame.kits.is_template IS
    'IQ.8: When true, kit appears in New-from-template picker (not a normal course kit).';
COMMENT ON COLUMN quizgame.kits.catalog_status IS
    'IQ.8: Public catalog moderation — unlisted|pending|listed|rejected.';
COMMENT ON COLUMN quizgame.kits.derived_from_kit_id IS
    'IQ.8: Provenance link to the kit this was duplicated/imported from.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_iq_public_kit_catalog BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_iq_public_kit_catalog IS
    'IQ.8: Enable curated public live-quiz kit catalog. Default OFF; org sharing works without it.';

-- Built-in starter templates (system scope). Stable IDs for idempotent seeds.
INSERT INTO quizgame.kits (
    id, course_id, title, description, slug, status, visibility, tags,
    is_template, template_scope, subject, grade_band, language, catalog_status
) VALUES
(
    'b1000000-0000-4000-8000-000000000001',
    NULL,
    'Exit ticket',
    'Quick end-of-class check for understanding.',
    'exit-ticket',
    'ready',
    'public',
    ARRAY['exit-ticket', 'formative', 'starter'],
    TRUE,
    'system',
    'General',
    'all',
    'en',
    'unlisted'
),
(
    'b1000000-0000-4000-8000-000000000002',
    NULL,
    'Team review',
    'Collaborative review game for teams.',
    'team-review',
    'ready',
    'public',
    ARRAY['team', 'review', 'starter'],
    TRUE,
    'system',
    'General',
    'all',
    'en',
    'unlisted'
),
(
    'b1000000-0000-4000-8000-000000000003',
    NULL,
    'Vocabulary race',
    'Fast-paced vocabulary practice with short timers.',
    'vocabulary-race',
    'ready',
    'public',
    ARRAY['vocabulary', 'language', 'starter'],
    TRUE,
    'system',
    'Language',
    'all',
    'en',
    'unlisted'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO quizgame.questions (
    id, kit_id, position, question_type, prompt, options, correct_answer,
    time_limit_seconds, points_style, answer_shuffle, explanation
) VALUES
(
    'b1000000-0000-4000-8000-000000000011',
    'b1000000-0000-4000-8000-000000000001',
    0,
    'mc_single',
    'What is one thing you learned today?',
    '[{"id":"a","text":"A new concept"},{"id":"b","text":"A skill to practice"},{"id":"c","text":"I am still unsure"}]'::jsonb,
    '"a"'::jsonb,
    30,
    'standard',
    TRUE,
    'Use student responses to plan tomorrow.'
),
(
    'b1000000-0000-4000-8000-000000000012',
    'b1000000-0000-4000-8000-000000000001',
    1,
    'type_answer',
    'In one word, how confident do you feel about today''s topic?',
    '[]'::jsonb,
    NULL,
    20,
    'no_points',
    FALSE,
    NULL
),
(
    'b1000000-0000-4000-8000-000000000021',
    'b1000000-0000-4000-8000-000000000002',
    0,
    'true_false',
    'Teams should discuss before locking in an answer.',
    '[{"id":"t","text":"True"},{"id":"f","text":"False"}]'::jsonb,
    'true'::jsonb,
    20,
    'standard',
    FALSE,
    'Collaboration is the point of team review.'
),
(
    'b1000000-0000-4000-8000-000000000022',
    'b1000000-0000-4000-8000-000000000002',
    1,
    'mc_multiple',
    'Which habits help a team succeed?',
    '[{"id":"a","text":"Listen to every member"},{"id":"b","text":"Rush the first idea"},{"id":"c","text":"Check the prompt carefully"}]'::jsonb,
    '["a","c"]'::jsonb,
    25,
    'double',
    TRUE,
    NULL
),
(
    'b1000000-0000-4000-8000-000000000031',
    'b1000000-0000-4000-8000-000000000003',
    0,
    'type_answer',
    'Type a synonym for "rapid".',
    '[]'::jsonb,
    '"fast"'::jsonb,
    15,
    'standard',
    FALSE,
    'Accept close synonyms when hosting live.'
),
(
    'b1000000-0000-4000-8000-000000000032',
    'b1000000-0000-4000-8000-000000000003',
    1,
    'mc_single',
    'Which word means "to say something again"?',
    '[{"id":"a","text":"Repeat"},{"id":"b","text":"Ignore"},{"id":"c","text":"Forget"}]'::jsonb,
    '"a"'::jsonb,
    12,
    'standard',
    TRUE,
    NULL
)
ON CONFLICT (id) DO NOTHING;
