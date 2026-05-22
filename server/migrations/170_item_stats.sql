-- Item analysis: per-question CTT statistics and test-level reliability coefficients (plan 9.4).

CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.item_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id         UUID NOT NULL,
    question_index  INTEGER NOT NULL,
    question_text   TEXT,
    n_responses     INTEGER NOT NULL,
    p_value         REAL,
    r_pb            REAL,
    distractor_freqs JSONB,
    flag            TEXT CHECK (flag IN ('easy', 'hard', 'poor_discriminator')),
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (quiz_id, question_index, computed_at)
);

CREATE INDEX IF NOT EXISTS idx_item_stats_quiz ON analytics.item_stats (quiz_id, computed_at DESC);

CREATE TABLE IF NOT EXISTS analytics.test_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id         UUID NOT NULL UNIQUE,
    n_responses     INTEGER NOT NULL,
    kr20            REAL,
    cronbach_alpha  REAL,
    mean_score      REAL,
    std_dev         REAL,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE analytics.item_stats IS
    'Per-question classical test theory statistics (p-value, point-biserial, distractor freqs).';
COMMENT ON TABLE analytics.test_stats IS
    'Test-level reliability coefficients (KR-20, Cronbach alpha) for a quiz administration.';
