CREATE TABLE IF NOT EXISTS marketing.content_build_settings (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    provider text NOT NULL DEFAULT 'none' CHECK (provider IN ('none', 'github')),
    repository text NOT NULL DEFAULT '',
    workflow_ref text NOT NULL DEFAULT 'main',
    token_ciphertext bytea,
    quiet_seconds integer NOT NULL DEFAULT 180 CHECK (quiet_seconds BETWEEN 0 AND 900),
    max_wait_seconds integer NOT NULL DEFAULT 900 CHECK (max_wait_seconds BETWEEN 1 AND 3600),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO marketing.content_build_settings (singleton) VALUES (true) ON CONFLICT (singleton) DO NOTHING;

CREATE TABLE IF NOT EXISTS marketing.content_builds (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','dispatched','running','succeeded','failed','timed_out')),
    reason text NOT NULL CHECK (reason IN ('publish','unpublish','archive','update','manual','scheduled')),
    paths text[] NOT NULL DEFAULT '{}',
    urgent boolean NOT NULL DEFAULT false,
    not_before timestamptz NOT NULL DEFAULT now(),
    deadline timestamptz NOT NULL,
    dispatched_at timestamptz,
    completed_at timestamptz,
    provider_run_id text,
    provider_run_url text,
    error text,
    requested_by uuid REFERENCES "user".users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS marketing_content_one_pending_build ON marketing.content_builds ((status)) WHERE status='pending';
CREATE INDEX IF NOT EXISTS marketing_content_builds_recent ON marketing.content_builds (created_at DESC);

CREATE TABLE IF NOT EXISTS marketing.content_publish_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id uuid REFERENCES marketing.content_articles(id) ON DELETE SET NULL,
    path text NOT NULL,
    action text NOT NULL CHECK (action IN ('publish','unpublish','archive','update','schedule','scheduled_publish','restore')),
    actor_id uuid REFERENCES "user".users(id) ON DELETE SET NULL,
    build_id uuid REFERENCES marketing.content_builds(id) ON DELETE SET NULL,
    error text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS marketing_content_publish_events_recent ON marketing.content_publish_events (created_at DESC);
CREATE INDEX IF NOT EXISTS marketing_content_publish_events_article ON marketing.content_publish_events (article_id, created_at DESC);
