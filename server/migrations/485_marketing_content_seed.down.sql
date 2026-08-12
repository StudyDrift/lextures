-- Remove only rows created by migration 485. Existing/human-authored rows are preserved.
DELETE FROM marketing.content_articles WHERE extra->>'seedMigration' = '485';
DELETE FROM marketing.content_categories c
WHERE c.locale = 'en'
  AND c.id::text = md5('mc-category:en:'||c.slug)::uuid::text
  AND NOT EXISTS (SELECT 1 FROM marketing.content_articles a WHERE a.category_id = c.id);
DELETE FROM marketing.content_authors a
WHERE a.slug = 'chase-willden'
  AND NOT EXISTS (SELECT 1 FROM marketing.content_articles c WHERE c.author_slug = a.slug OR c.reviewer_slug = a.slug);
