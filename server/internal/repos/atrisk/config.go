package atrisk

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const configSelectCols = `
threshold, weight_missing, weight_quiz, weight_inactive, weight_trend,
quiz_avg_threshold, inactive_days_threshold, missing_pct_threshold
`

// Config holds at-risk scoring parameters.
type Config struct {
	OrgID                 uuid.UUID
	Threshold             float32
	WeightMissing         float32
	WeightQuiz            float32
	WeightInactive        float32
	WeightTrend           float32
	QuizAvgThreshold      float32
	InactiveDaysThreshold int
	MissingPctThreshold   float32
}

// DefaultConfig returns platform defaults when no row exists.
func DefaultConfig(orgID uuid.UUID) Config {
	return Config{
		OrgID:                 orgID,
		Threshold:             60,
		WeightMissing:         0.35,
		WeightQuiz:            0.25,
		WeightInactive:        0.25,
		WeightTrend:           0.15,
		QuizAvgThreshold:      60,
		InactiveDaysThreshold: 7,
		MissingPctThreshold:   100,
	}
}

// ValidateConfig checks numeric bounds and weight sum.
func ValidateConfig(c Config) error {
	sum := c.WeightMissing + c.WeightQuiz + c.WeightInactive + c.WeightTrend
	if sum < 0.999 || sum > 1.001 {
		return errors.New("weights must sum to 1")
	}
	if c.Threshold < 0 || c.Threshold > 100 {
		return errors.New("threshold out of range")
	}
	if c.QuizAvgThreshold < 0 || c.QuizAvgThreshold > 100 {
		return errors.New("quizAvgThreshold out of range")
	}
	if c.MissingPctThreshold < 1 || c.MissingPctThreshold > 100 {
		return errors.New("missingPctThreshold out of range")
	}
	if c.InactiveDaysThreshold < 1 || c.InactiveDaysThreshold > 90 {
		return errors.New("inactiveDaysThreshold out of range")
	}
	return nil
}

// LoadEffective returns tenant config or defaults.
func LoadEffective(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (Config, error) {
	var c Config
	err := pool.QueryRow(ctx, `
SELECT org_id, `+configSelectCols+`
FROM analytics.at_risk_config
WHERE org_id = $1
`, orgID).Scan(
		&c.OrgID, &c.Threshold, &c.WeightMissing, &c.WeightQuiz, &c.WeightInactive, &c.WeightTrend,
		&c.QuizAvgThreshold, &c.InactiveDaysThreshold, &c.MissingPctThreshold,
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
	if err := ValidateConfig(c); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.at_risk_config (
    org_id, threshold, weight_missing, weight_quiz, weight_inactive, weight_trend,
    quiz_avg_threshold, inactive_days_threshold, missing_pct_threshold, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (org_id) DO UPDATE SET
    threshold = EXCLUDED.threshold,
    weight_missing = EXCLUDED.weight_missing,
    weight_quiz = EXCLUDED.weight_quiz,
    weight_inactive = EXCLUDED.weight_inactive,
    weight_trend = EXCLUDED.weight_trend,
    quiz_avg_threshold = EXCLUDED.quiz_avg_threshold,
    inactive_days_threshold = EXCLUDED.inactive_days_threshold,
    missing_pct_threshold = EXCLUDED.missing_pct_threshold,
    updated_at = now()
`, c.OrgID, c.Threshold, c.WeightMissing, c.WeightQuiz, c.WeightInactive, c.WeightTrend,
		c.QuizAvgThreshold, c.InactiveDaysThreshold, c.MissingPctThreshold)
	return err
}

// LoadEffectiveForCourse returns course override, else tenant config, else defaults.
// The bool is true when a course-specific row exists.
func LoadEffectiveForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (Config, bool, error) {
	var c Config
	err := pool.QueryRow(ctx, `
SELECT `+configSelectCols+`
FROM analytics.course_at_risk_config
WHERE course_id = $1
`, courseID).Scan(
		&c.Threshold, &c.WeightMissing, &c.WeightQuiz, &c.WeightInactive, &c.WeightTrend,
		&c.QuizAvgThreshold, &c.InactiveDaysThreshold, &c.MissingPctThreshold,
	)
	if err == nil {
		return c, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Config{}, false, err
	}
	var orgID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID)
	if err != nil {
		return Config{}, false, err
	}
	cfg, err := LoadEffective(ctx, pool, orgID)
	return cfg, false, err
}

// UpsertCourse saves per-course at-risk configuration.
func UpsertCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, c Config) error {
	if err := ValidateConfig(c); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.course_at_risk_config (
    course_id, threshold, weight_missing, weight_quiz, weight_inactive, weight_trend,
    quiz_avg_threshold, inactive_days_threshold, missing_pct_threshold, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (course_id) DO UPDATE SET
    threshold = EXCLUDED.threshold,
    weight_missing = EXCLUDED.weight_missing,
    weight_quiz = EXCLUDED.weight_quiz,
    weight_inactive = EXCLUDED.weight_inactive,
    weight_trend = EXCLUDED.weight_trend,
    quiz_avg_threshold = EXCLUDED.quiz_avg_threshold,
    inactive_days_threshold = EXCLUDED.inactive_days_threshold,
    missing_pct_threshold = EXCLUDED.missing_pct_threshold,
    updated_at = now()
`, courseID, c.Threshold, c.WeightMissing, c.WeightQuiz, c.WeightInactive, c.WeightTrend,
		c.QuizAvgThreshold, c.InactiveDaysThreshold, c.MissingPctThreshold)
	return err
}
