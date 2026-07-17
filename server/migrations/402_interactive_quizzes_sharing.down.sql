ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_iq_public_kit_catalog;

DROP INDEX IF EXISTS idx_quizgame_kits_language;
DROP INDEX IF EXISTS idx_quizgame_kits_grade_band;
DROP INDEX IF EXISTS idx_quizgame_kits_subject;
DROP INDEX IF EXISTS idx_quizgame_kits_title_fts;
DROP INDEX IF EXISTS idx_quizgame_kits_templates;
DROP INDEX IF EXISTS idx_quizgame_kits_catalog;
DROP INDEX IF EXISTS idx_quizgame_kit_shares_grantee;
DROP INDEX IF EXISTS idx_quizgame_kit_shares_kit;
DROP INDEX IF EXISTS idx_quizgame_kit_shares_unique;

DROP TABLE IF EXISTS quizgame.kit_shares;

DELETE FROM quizgame.questions WHERE kit_id IN (
    'b1000000-0000-4000-8000-000000000001',
    'b1000000-0000-4000-8000-000000000002',
    'b1000000-0000-4000-8000-000000000003'
);
DELETE FROM quizgame.kits WHERE id IN (
    'b1000000-0000-4000-8000-000000000001',
    'b1000000-0000-4000-8000-000000000002',
    'b1000000-0000-4000-8000-000000000003'
);

ALTER TABLE quizgame.kits DROP CONSTRAINT IF EXISTS quizgame_kits_course_or_system_template_chk;
ALTER TABLE quizgame.kits DROP CONSTRAINT IF EXISTS quizgame_kits_catalog_status_chk;
ALTER TABLE quizgame.kits DROP CONSTRAINT IF EXISTS quizgame_kits_template_scope_chk;

ALTER TABLE quizgame.kits
    DROP COLUMN IF EXISTS catalog_status,
    DROP COLUMN IF EXISTS language,
    DROP COLUMN IF EXISTS grade_band,
    DROP COLUMN IF EXISTS subject,
    DROP COLUMN IF EXISTS attribution,
    DROP COLUMN IF EXISTS derived_from_kit_id,
    DROP COLUMN IF EXISTS template_scope,
    DROP COLUMN IF EXISTS is_template;

-- Restore NOT NULL only when every remaining row has a course_id.
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM quizgame.kits WHERE course_id IS NULL) THEN
        ALTER TABLE quizgame.kits ALTER COLUMN course_id SET NOT NULL;
    END IF;
END $$;
