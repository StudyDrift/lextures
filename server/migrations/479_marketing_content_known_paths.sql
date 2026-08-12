CREATE TABLE IF NOT EXISTS marketing.content_known_paths (
    path TEXT PRIMARY KEY,
    source TEXT NOT NULL CHECK (source IN ('article', 'static_route')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mc_known_paths_source ON marketing.content_known_paths(source);

INSERT INTO marketing.content_known_paths(path, source)
SELECT path, 'article' FROM marketing.content_articles
WHERE status = 'published' AND deleted_at IS NULL
ON CONFLICT (path) DO UPDATE SET source = 'article', updated_at = now();

CREATE OR REPLACE FUNCTION marketing.sync_content_article_path() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM marketing.content_known_paths WHERE source = 'article' AND path = OLD.path;
        RETURN OLD;
    END IF;
    IF NEW.status <> 'published' OR NEW.deleted_at IS NOT NULL THEN
        DELETE FROM marketing.content_known_paths WHERE source = 'article' AND path = COALESCE(OLD.path, NEW.path);
        RETURN NEW;
    END IF;
    INSERT INTO marketing.content_known_paths(path, source, updated_at)
    VALUES (NEW.path, 'article', now())
    ON CONFLICT (path) DO UPDATE SET source = 'article', updated_at = now();
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_mc_article_known_path ON marketing.content_articles;
CREATE TRIGGER trg_mc_article_known_path AFTER INSERT OR UPDATE OR DELETE
ON marketing.content_articles FOR EACH ROW EXECUTE FUNCTION marketing.sync_content_article_path();
