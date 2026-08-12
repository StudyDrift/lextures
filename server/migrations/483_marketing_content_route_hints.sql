-- MC.13 — docs search & in-app help integration: route hints and search-query logging.

CREATE TABLE IF NOT EXISTS marketing.content_route_hints (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_prefix TEXT NOT NULL,
    article_id   UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    position     INTEGER NOT NULL DEFAULT 100,
    created_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (route_prefix, article_id)
);
CREATE INDEX IF NOT EXISTS idx_mc_route_hints_prefix ON marketing.content_route_hints (route_prefix);

CREATE TABLE IF NOT EXISTS marketing.content_search_queries (
    day        DATE NOT NULL,
    query      TEXT NOT NULL,
    surface    TEXT NOT NULL CHECK (surface IN ('widget', 'www', 'workspace')),
    hits       INTEGER NOT NULL DEFAULT 0,
    results    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (day, query, surface)
);

-- Backfill: seed route hints from the legacy static mapping in support_widget_http.go
-- so day-one behaviour is at least as good as today. Matches articles by slug; any
-- slug not yet present in the content table is skipped (no-op on empty catalogs).
INSERT INTO marketing.content_route_hints (route_prefix, article_id, position)
SELECT m.prefix, a.id, m.position
FROM (VALUES
    ('/', 'finding-your-course', 1),
    ('/dashboard', 'finding-your-course', 1),
    ('/dashboard', 'navigating-the-course-interface', 2),
    ('/courses', 'finding-your-course', 1),
    ('/courses', 'navigating-the-course-interface', 2),
    ('/courses', 'creating-a-new-course', 3),
    ('/quiz', 'navigating-the-course-interface', 1),
    ('/gradebook', 'navigating-the-course-interface', 1),
    ('/settings', 'finding-your-course', 1),
    ('/inbox', 'navigating-the-course-interface', 1)
) AS m(prefix, slug, position)
JOIN marketing.content_articles a ON a.slug = m.slug AND a.kind = 'doc' AND a.deleted_at IS NULL
ON CONFLICT (route_prefix, article_id) DO NOTHING;
