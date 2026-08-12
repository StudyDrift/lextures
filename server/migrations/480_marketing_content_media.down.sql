ALTER TABLE marketing.content_authors DROP CONSTRAINT IF EXISTS fk_mc_authors_image_media;
ALTER TABLE marketing.content_articles DROP CONSTRAINT IF EXISTS fk_mc_articles_hero_media;
DROP TABLE IF EXISTS marketing.content_article_media;
DROP TABLE IF EXISTS marketing.content_media;
