-- MC.11 — editorial workflow, planning, freshness and governance.

ALTER TABLE marketing.content_articles
    ADD COLUMN IF NOT EXISTS reviewer_id UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS review_submitted_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS marketing.content_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES marketing.content_articles(id) ON DELETE CASCADE,
    revision_no INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('submitted','approved','changes_requested','reviewed')),
    reviewer_id UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    actor_id UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_mc_reviews_article ON marketing.content_reviews(article_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mc_review_queue ON marketing.content_articles(review_submitted_at, id)
    WHERE status='in_review' AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS marketing.content_briefs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('blog','doc')),
    pillar TEXT NOT NULL DEFAULT '', cluster TEXT NOT NULL DEFAULT '',
    primary_question TEXT NOT NULL DEFAULT '',
    owner_id UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    target_date DATE, brief_ref TEXT NOT NULL DEFAULT '',
    article_id UUID REFERENCES marketing.content_articles(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned','in_progress','published','dropped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_mc_briefs_calendar ON marketing.content_briefs(target_date, id);
DROP TRIGGER IF EXISTS content_briefs_updated_at ON marketing.content_briefs;
CREATE TRIGGER content_briefs_updated_at BEFORE UPDATE ON marketing.content_briefs
FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();

CREATE TABLE IF NOT EXISTS marketing.content_link_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES marketing.content_articles(id) ON DELETE CASCADE,
    url TEXT NOT NULL, status_code INTEGER, error TEXT,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(article_id,url)
);
CREATE TABLE IF NOT EXISTS marketing.content_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES marketing.content_articles(id) ON DELETE CASCADE,
    revision_no INTEGER NOT NULL,
    actor_id UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    rules TEXT[] NOT NULL, justification TEXT NOT NULL CHECK (length(trim(justification)) >= 10),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_mc_overrides_recent ON marketing.content_overrides(created_at DESC);
CREATE TABLE IF NOT EXISTS marketing.content_health_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), taken_at TIMESTAMPTZ NOT NULL DEFAULT now(), payload JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mc_health_snapshots_recent ON marketing.content_health_snapshots(taken_at DESC);

CREATE TABLE IF NOT EXISTS marketing.content_editorial_settings (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton),
    review_interval_doc_days INTEGER NOT NULL DEFAULT 180 CHECK(review_interval_doc_days > 0),
    review_interval_blog_days INTEGER NOT NULL DEFAULT 365 CHECK(review_interval_blog_days > 0),
    stale_threshold_pct NUMERIC(5,2) NOT NULL DEFAULT 10 CHECK(stale_threshold_pct BETWEEN 0 AND 100),
    revision_retention_months INTEGER NOT NULL DEFAULT 18 CHECK(revision_retention_months >= 1),
    pillars JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO marketing.content_editorial_settings(singleton,pillars) VALUES (true, '[
 {"id":"p1","slug":"adaptive-learning","title":"Adaptive learning: how it actually works","floor":18},
 {"id":"p2","slug":"assessment-design-ai","title":"Assessment design in the age of generative AI","floor":20},
 {"id":"p3","slug":"grading-and-integrity","title":"Grading, feedback and academic integrity","floor":16},
 {"id":"p4","slug":"mastery-and-standards","title":"Standards, outcomes and mastery-based grading","floor":14},
 {"id":"p5","slug":"choosing-an-lms","title":"Choosing and running a learning platform","floor":16},
 {"id":"p6","slug":"homeschool-teaching","title":"Teaching at home: curriculum, pacing and evidence","floor":12}
]') ON CONFLICT(singleton) DO NOTHING;

CREATE TABLE IF NOT EXISTS marketing.content_notification_log (
    dedupe_key TEXT PRIMARY KEY, article_id UUID REFERENCES marketing.content_articles(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL, recipient_id UUID REFERENCES "user".users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Imported published content receives a useful due date on day one.
UPDATE marketing.content_articles a SET review_due_on =
    COALESCE(a.content_updated_at::date, a.published_at::date, CURRENT_DATE) +
    CASE WHEN a.kind='doc' THEN (SELECT review_interval_doc_days FROM marketing.content_editorial_settings WHERE singleton)
         ELSE (SELECT review_interval_blog_days FROM marketing.content_editorial_settings WHERE singleton) END
WHERE a.status='published' AND a.review_due_on IS NULL;
