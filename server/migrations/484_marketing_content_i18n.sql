-- MC.14 — localization, translation groups, and locale-aware full-text search.

ALTER TABLE marketing.content_articles
    ADD COLUMN IF NOT EXISTS source_article_id UUID REFERENCES marketing.content_articles (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS source_synced_revision INTEGER,
    ADD COLUMN IF NOT EXISTS source_synced_at TIMESTAMPTZ;

ALTER TABLE marketing.content_categories
    ADD COLUMN IF NOT EXISTS category_group_id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE INDEX IF NOT EXISTS idx_mc_categories_group ON marketing.content_categories (category_group_id);
CREATE INDEX IF NOT EXISTS idx_mc_articles_source ON marketing.content_articles (source_article_id)
    WHERE source_article_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS marketing.content_locales (
    code        TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    rtl         BOOLEAN NOT NULL DEFAULT FALSE,
    ts_config   TEXT NOT NULL DEFAULT 'simple',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INTEGER NOT NULL DEFAULT 100,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_mc_locales_one_default
    ON marketing.content_locales ((is_default)) WHERE is_default;
DROP TRIGGER IF EXISTS content_locales_updated_at ON marketing.content_locales;
CREATE TRIGGER content_locales_updated_at BEFORE UPDATE ON marketing.content_locales
    FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();

INSERT INTO marketing.content_locales (code, label, is_default, rtl, ts_config, enabled, sort_order)
VALUES
    ('en', 'English',  TRUE,  FALSE, 'english', TRUE,  10),
    ('es', 'Español',  FALSE, FALSE, 'spanish', FALSE, 20),
    ('fr', 'Français', FALSE, FALSE, 'french',  FALSE, 30),
    ('ar', 'العربية',  FALSE, TRUE,  'simple',  FALSE, 40)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE marketing.content_editorial_settings
    ADD COLUMN IF NOT EXISTS locales_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Locale is immutable after insert (translations are separate rows).
CREATE OR REPLACE FUNCTION marketing.prevent_article_locale_change()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.locale IS DISTINCT FROM OLD.locale THEN
        RAISE EXCEPTION 'marketing content locale is immutable'
            USING ERRCODE = '22023';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS content_articles_locale_immutable ON marketing.content_articles;
CREATE TRIGGER content_articles_locale_immutable
    BEFORE UPDATE ON marketing.content_articles
    FOR EACH ROW EXECUTE FUNCTION marketing.prevent_article_locale_change();

-- Generated search_tsv cannot reference content_locales; replace with a trigger.
ALTER TABLE marketing.content_articles DROP COLUMN IF EXISTS search_tsv;
ALTER TABLE marketing.content_articles ADD COLUMN search_tsv TSVECTOR;

CREATE OR REPLACE FUNCTION marketing.content_articles_search_tsv()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    cfg regconfig := 'simple';
BEGIN
    SELECT COALESCE(l.ts_config, 'simple')::regconfig INTO cfg
    FROM marketing.content_locales l
    WHERE l.code = NEW.locale;
    IF cfg IS NULL THEN
        cfg := 'simple';
    END IF;
    NEW.search_tsv :=
        setweight(to_tsvector(cfg, coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector(cfg, coalesce(NEW.description, '')), 'B') ||
        setweight(to_tsvector(cfg, coalesce(NEW.primary_question, '')), 'B') ||
        setweight(to_tsvector(cfg, marketing.text_array_to_string(NEW.keywords)), 'C') ||
        setweight(to_tsvector(cfg, coalesce(NEW.body_md, '')), 'D');
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS content_articles_search_tsv ON marketing.content_articles;
CREATE TRIGGER content_articles_search_tsv
    BEFORE INSERT OR UPDATE OF title, description, primary_question, keywords, body_md, locale
    ON marketing.content_articles
    FOR EACH ROW EXECUTE FUNCTION marketing.content_articles_search_tsv();

UPDATE marketing.content_articles
SET title = title;

CREATE INDEX IF NOT EXISTS idx_mc_articles_search ON marketing.content_articles USING GIN (search_tsv);
