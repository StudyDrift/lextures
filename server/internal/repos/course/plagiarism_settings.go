package course

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlagiarismSettingsJSON is the course-level plagiarism workflow config (plan 14.8).
type PlagiarismSettingsJSON struct {
	PlagiarismChecksEnabled     bool     `json:"plagiarismChecksEnabled"`
	PlagiarismProvider          *string  `json:"plagiarismProvider"`
	PlagiarismAlertThresholdPct float64  `json:"plagiarismAlertThresholdPct"`
}

// PlagiarismSettingsPatch is a partial update payload.
type PlagiarismSettingsPatch struct {
	ChecksEnabled     *bool
	Provider          *string
	AlertThresholdPct *float64
}

// GetPlagiarismSettings loads plagiarism settings for a course.
func GetPlagiarismSettings(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*PlagiarismSettingsJSON, error) {
	var out PlagiarismSettingsJSON
	var provider sql.NullString
	err := pool.QueryRow(ctx, `
SELECT COALESCE(plagiarism_checks_enabled, true),
       plagiarism_provider,
       plagiarism_alert_threshold_pct::float8
FROM course.courses
WHERE course_code = $1
`, courseCode).Scan(&out.PlagiarismChecksEnabled, &provider, &out.PlagiarismAlertThresholdPct)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if provider.Valid {
		v := provider.String
		out.PlagiarismProvider = &v
	}
	return &out, nil
}

// PatchPlagiarismSettings updates course plagiarism settings.
func PatchPlagiarismSettings(ctx context.Context, pool *pgxpool.Pool, courseCode string, patch PlagiarismSettingsPatch) (*PlagiarismSettingsJSON, error) {
	cur, err := GetPlagiarismSettings(ctx, pool, courseCode)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, nil
	}
	enabled := cur.PlagiarismChecksEnabled
	if patch.ChecksEnabled != nil {
		enabled = *patch.ChecksEnabled
	}
	provider := cur.PlagiarismProvider
	if patch.Provider != nil {
		if *patch.Provider == "" {
			provider = nil
		} else {
			provider = patch.Provider
		}
	}
	threshold := cur.PlagiarismAlertThresholdPct
	if patch.AlertThresholdPct != nil {
		threshold = *patch.AlertThresholdPct
	}
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	_, err = pool.Exec(ctx, `
UPDATE course.courses
SET plagiarism_checks_enabled = $2,
    plagiarism_provider = $3,
    plagiarism_alert_threshold_pct = $4
WHERE course_code = $1
`, courseCode, enabled, provider, threshold)
	if err != nil {
		return nil, err
	}
	return GetPlagiarismSettings(ctx, pool, courseCode)
}
