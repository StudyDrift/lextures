-- Optional course allowlist for personal access keys (empty = all accessible courses).

ALTER TABLE auth.api_tokens
    ADD COLUMN IF NOT EXISTS course_ids UUID[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN auth.api_tokens.course_ids IS
    'When non-empty, the key may only access these course IDs. Empty means all courses the owner can access.';
