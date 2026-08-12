DROP TRIGGER IF EXISTS content_articles_search_tsv ON marketing.content_articles;
DROP FUNCTION IF EXISTS marketing.content_articles_search_tsv();
DROP TRIGGER IF EXISTS content_articles_locale_immutable ON marketing.content_articles;
DROP FUNCTION IF EXISTS marketing.prevent_article_locale_change();

ALTER TABLE marketing.content_articles DROP COLUMN IF EXISTS search_tsv;
ALTER TABLE marketing.content_articles
    ADD COLUMN search_tsv TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(primary_question, '')), 'B') ||
        setweight(to_tsvector('english', marketing.text_array_to_string(keywords)), 'C') ||
        setweight(to_tsvector('english', coalesce(body_md, '')), 'D')
    ) STORED;
CREATE INDEX IF NOT EXISTS idx_mc_articles_search ON marketing.content_articles USING GIN (search_tsv);

ALTER TABLE marketing.content_editorial_settings DROP COLUMN IF EXISTS locales_enabled;
DROP TRIGGER IF EXISTS content_locales_updated_at ON marketing.content_locales;
DROP TABLE IF EXISTS marketing.content_locales;
DROP INDEX IF EXISTS idx_mc_articles_source;
DROP INDEX IF EXISTS idx_mc_categories_group;
ALTER TABLE marketing.content_categories DROP COLUMN IF EXISTS category_group_id;
ALTER TABLE marketing.content_articles
    DROP COLUMN IF EXISTS source_synced_at,
    DROP COLUMN IF EXISTS source_synced_revision,
    DROP COLUMN IF EXISTS source_article_id;
