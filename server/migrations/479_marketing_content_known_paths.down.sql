DROP TRIGGER IF EXISTS trg_mc_article_known_path ON marketing.content_articles;
DROP FUNCTION IF EXISTS marketing.sync_content_article_path();
DROP TABLE IF EXISTS marketing.content_known_paths;
