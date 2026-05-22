package atrisk

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds tenant-scored at-risk parameters.
type Config struct {
	OrgID            uuid.UUID
	Threshold        float32
	WeightMissing    float32
	WeightQuiz       float32
	WeightInactive   float32
	WeightTrend      float32
	QuizAvgThreshold float32
}

// DefaultConfig returns platform defaults when no row exists.
func DefaultConfig(orgID uuid.UUID) Config {
	return Config{
		OrgID:            orgID,
		Threshold:        60,
		WeightMissing:    0.35,
		WeightQuiz:       0.25,
		WeightInactive:   0.25,
		WeightTrend:      0.15,
		QuizAvgThreshold: 60,
	}
}

// LoadEffective returns tenant config or defaults.
func LoadEffective(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (Config, error) {
	var c Config
	err := pool.QueryRow(ctx, `
SELECT org_id, threshold, weight_missing, weight_quiz, weight_inactive, weight_trend, quiz_avg_threshold
FROM analytics.at_risk_config
WHERE org_id = $1
`, orgID).Scan(
		&c.OrgID, &c.Threshold, &c.WeightMissing, &c.WeightQuiz, &c.WeightInactive, &c.WeightTrend, &c.QuizAvgThreshold,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultConfig(orgID), nil
	}
	if err != nil {
		return Config{}, err
	}
	return c, nil
}

// Upsert saves tenant at-risk configuration.
func Upsert(ctx context.Context, pool *pgxpool.Pool, c Config) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.at_risk_config (
    org_id, threshold, weight_missing, weight_quiz, weight_inactive, weight_trend, quiz_avg_threshold, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (org_id) DO UPDATE SET
    threshold = EXCLUDED.threshold,
    weight_missing = EXCLUDED.weight_missing,
    weight_quiz = EXCLUDED.weight_quiz,
    weight_inactive = EXCLUDED.weight_inactive,
    weight_trend = EXCLUDED.weight_trend,
    quiz_avg_threshold = EXCLUDED.quiz_avg_threshold,
    updated_at = now()
`, c.OrgID, c.Threshold, c.WeightMissing, c.WeightQuiz, c.WeightInactive, c.WeightTrend, c.QuizAvgThreshold)
	return err
}
