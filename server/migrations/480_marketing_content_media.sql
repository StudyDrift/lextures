-- MC.5 — reusable, content-addressed marketing media.
CREATE TABLE marketing.content_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checksum TEXT NOT NULL UNIQUE,
    mime_type TEXT NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size > 0),
    width INTEGER,
    height INTEGER,
    alt_text TEXT NOT NULL,
    decorative BOOLEAN NOT NULL DEFAULT FALSE,
    title TEXT NOT NULL DEFAULT '',
    credit TEXT NOT NULL DEFAULT '',
    storage_key TEXT NOT NULL,
    renditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    uploaded_by UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT content_media_alt_or_decorative CHECK (decorative OR length(btrim(alt_text)) > 0)
);
CREATE INDEX idx_mc_media_created ON marketing.content_media(created_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE marketing.content_article_media (
    article_id UUID NOT NULL REFERENCES marketing.content_articles(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES marketing.content_media(id),
    usage TEXT NOT NULL CHECK (usage IN ('body','hero')),
    PRIMARY KEY(article_id, media_id, usage)
);

ALTER TABLE marketing.content_articles ADD CONSTRAINT fk_mc_articles_hero_media
    FOREIGN KEY(hero_media_id) REFERENCES marketing.content_media(id);
ALTER TABLE marketing.content_authors ADD CONSTRAINT fk_mc_authors_image_media
    FOREIGN KEY(image_media_id) REFERENCES marketing.content_media(id);

