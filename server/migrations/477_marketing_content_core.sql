-- MC.1 — database-backed blog and help-center content.

CREATE SCHEMA IF NOT EXISTS marketing;

CREATE OR REPLACE FUNCTION marketing.set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

-- array_to_string is stable in PostgreSQL's catalog because generic type output
-- can be locale-sensitive. This text-only wrapper is safe to mark immutable and
-- permits the required generated search vector.
CREATE OR REPLACE FUNCTION marketing.text_array_to_string(values_to_join TEXT[])
RETURNS TEXT LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT array_to_string(values_to_join, ' ')
$$;

CREATE TABLE IF NOT EXISTS marketing.content_authors (
    slug TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    job_title TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    knows_about TEXT[] NOT NULL DEFAULT '{}',
    image_media_id UUID,
    links JSONB NOT NULL DEFAULT '{}'::jsonb,
    user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'retired')),
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS marketing.content_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'en',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 100,
    platform_path TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (locale, slug)
);

CREATE TABLE IF NOT EXISTS marketing.content_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind TEXT NOT NULL CHECK (kind IN ('blog', 'doc')),
    slug TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'en',
    translation_group_id UUID NOT NULL DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES marketing.content_categories (id),
    path TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    body_md TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','in_review','changes_requested','scheduled','published','archived')),
    author_slug TEXT NOT NULL REFERENCES marketing.content_authors (slug),
    reviewer_slug TEXT REFERENCES marketing.content_authors (slug),
    published_at TIMESTAMPTZ,
    first_published_at TIMESTAMPTZ,
    scheduled_for TIMESTAMPTZ,
    content_updated_at TIMESTAMPTZ,
    reviewed_at DATE,
    review_due_on DATE,
    primary_question TEXT NOT NULL DEFAULT '',
    cluster TEXT NOT NULL DEFAULT '',
    pillar TEXT NOT NULL DEFAULT '',
    brief_ref TEXT NOT NULL DEFAULT '',
    verified_against TEXT NOT NULL DEFAULT '',
    keywords TEXT[] NOT NULL DEFAULT '{}',
    related_to TEXT[] NOT NULL DEFAULT '{}',
    roles TEXT[] NOT NULL DEFAULT '{}',
    segments TEXT[] NOT NULL DEFAULT '{}',
    citations TEXT[] NOT NULL DEFAULT '{}',
    hero_media_id UUID,
    quality_score NUMERIC(3,1),
    quality_report JSONB,
    noindex BOOLEAN NOT NULL DEFAULT FALSE,
    canonical_override TEXT,
    extra JSONB NOT NULL DEFAULT '{}'::jsonb,
    revision_no INTEGER NOT NULL DEFAULT 1 CHECK (revision_no > 0),
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    search_tsv TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(primary_question, '')), 'B') ||
        setweight(to_tsvector('english', marketing.text_array_to_string(keywords)), 'C') ||
        setweight(to_tsvector('english', coalesce(body_md, '')), 'D')
    ) STORED,
    CONSTRAINT marketing_doc_requires_category CHECK (kind <> 'doc' OR category_id IS NOT NULL),
    CONSTRAINT marketing_blog_has_no_category CHECK (kind <> 'blog' OR category_id IS NULL),
    CONSTRAINT marketing_published_has_timestamp CHECK (status <> 'published' OR published_at IS NOT NULL),
    CONSTRAINT marketing_scheduled_has_timestamp CHECK (status <> 'scheduled' OR scheduled_for IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mc_articles_slug_live ON marketing.content_articles (kind, locale, slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mc_articles_path_live ON marketing.content_articles (path) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mc_articles_admin_list ON marketing.content_articles (kind, status, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mc_articles_published ON marketing.content_articles (kind, locale, published_at DESC) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mc_articles_due_publish ON marketing.content_articles (scheduled_for) WHERE status = 'scheduled';
CREATE INDEX IF NOT EXISTS idx_mc_articles_search ON marketing.content_articles USING GIN (search_tsv);
CREATE INDEX IF NOT EXISTS idx_mc_articles_group ON marketing.content_articles (translation_group_id);

CREATE TABLE IF NOT EXISTS marketing.content_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    revision_no INTEGER NOT NULL,
    body_md TEXT NOT NULL,
    metadata JSONB NOT NULL,
    change_note TEXT NOT NULL DEFAULT '',
    status_after TEXT NOT NULL,
    actor_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (article_id, revision_no)
);
CREATE INDEX IF NOT EXISTS idx_mc_revisions_article ON marketing.content_revisions (article_id, revision_no DESC);

CREATE TABLE IF NOT EXISTS marketing.content_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS marketing.content_article_tags (
    article_id UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES marketing.content_tags (id) ON DELETE CASCADE,
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (article_id, tag_id)
);

CREATE TABLE IF NOT EXISTS marketing.content_redirects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_path TEXT NOT NULL UNIQUE,
    to_path TEXT NOT NULL,
    status_code INTEGER NOT NULL DEFAULT 301 CHECK (status_code IN (301, 302)),
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'slug_change')),
    article_id UUID REFERENCES marketing.content_articles (id) ON DELETE SET NULL,
    created_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT marketing_redirect_paths_differ CHECK (from_path <> to_path)
);

CREATE OR REPLACE FUNCTION marketing.reject_redirect_article_path()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM marketing.content_articles WHERE path = NEW.from_path AND deleted_at IS NULL) THEN
        RAISE EXCEPTION 'redirect source conflicts with a live article path: %', NEW.from_path USING ERRCODE = 'unique_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION marketing.reject_article_redirect_path()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.deleted_at IS NULL AND EXISTS (SELECT 1 FROM marketing.content_redirects WHERE from_path = NEW.path) THEN
        RAISE EXCEPTION 'article path conflicts with a redirect source: %', NEW.path USING ERRCODE = 'unique_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS content_redirects_reject_article_path ON marketing.content_redirects;
CREATE TRIGGER content_redirects_reject_article_path
BEFORE INSERT OR UPDATE OF from_path ON marketing.content_redirects
FOR EACH ROW EXECUTE FUNCTION marketing.reject_redirect_article_path();

DROP TRIGGER IF EXISTS content_articles_reject_redirect_path ON marketing.content_articles;
CREATE TRIGGER content_articles_reject_redirect_path
BEFORE INSERT OR UPDATE OF path, deleted_at ON marketing.content_articles
FOR EACH ROW EXECUTE FUNCTION marketing.reject_article_redirect_path();

DROP TRIGGER IF EXISTS content_authors_updated_at ON marketing.content_authors;
CREATE TRIGGER content_authors_updated_at BEFORE UPDATE ON marketing.content_authors FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();
DROP TRIGGER IF EXISTS content_categories_updated_at ON marketing.content_categories;
CREATE TRIGGER content_categories_updated_at BEFORE UPDATE ON marketing.content_categories FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();
DROP TRIGGER IF EXISTS content_articles_updated_at ON marketing.content_articles;
CREATE TRIGGER content_articles_updated_at BEFORE UPDATE ON marketing.content_articles FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();
DROP TRIGGER IF EXISTS content_tags_updated_at ON marketing.content_tags;
CREATE TRIGGER content_tags_updated_at BEFORE UPDATE ON marketing.content_tags FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();
DROP TRIGGER IF EXISTS content_redirects_updated_at ON marketing.content_redirects;
CREATE TRIGGER content_redirects_updated_at BEFORE UPDATE ON marketing.content_redirects FOR EACH ROW EXECUTE FUNCTION marketing.set_updated_at();
