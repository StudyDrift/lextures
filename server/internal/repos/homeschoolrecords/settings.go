package homeschoolrecords

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Settings is the family's school name and school-year calendar.
type Settings struct {
	ParentUserID         uuid.UUID
	SchoolName           string
	SchoolYearStartMonth int
	SchoolYearStartDay   int
	UpdatedAt            time.Time
}

const (
	DefaultSchoolYearStartMonth = 8
	DefaultSchoolYearStartDay   = 1
)

// DefaultSettings returns empty settings with the August 1 school-year start.
func DefaultSettings(parentID uuid.UUID) Settings {
	return Settings{
		ParentUserID:         parentID,
		SchoolName:           "",
		SchoolYearStartMonth: DefaultSchoolYearStartMonth,
		SchoolYearStartDay:   DefaultSchoolYearStartDay,
	}
}

// GetSettings loads settings for a parent, or defaults when none are stored yet.
func GetSettings(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID) (Settings, error) {
	var s Settings
	err := pool.QueryRow(ctx, `
SELECT parent_user_id, school_name, school_year_start_month, school_year_start_day, updated_at
FROM "user".homeschool_records_settings
WHERE parent_user_id = $1
`, parentID).Scan(&s.ParentUserID, &s.SchoolName, &s.SchoolYearStartMonth, &s.SchoolYearStartDay, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultSettings(parentID), nil
	}
	if err != nil {
		return Settings{}, err
	}
	return s, nil
}

// UpsertSettings saves family school name and school-year start.
func UpsertSettings(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID, schoolName string, startMonth, startDay int) (Settings, error) {
	if startMonth < 1 || startMonth > 12 {
		startMonth = DefaultSchoolYearStartMonth
	}
	if startDay < 1 || startDay > 31 {
		startDay = DefaultSchoolYearStartDay
	}
	schoolName = strings.TrimSpace(schoolName)
	if len(schoolName) > 200 {
		schoolName = schoolName[:200]
	}
	var s Settings
	err := pool.QueryRow(ctx, `
INSERT INTO "user".homeschool_records_settings (
    parent_user_id, school_name, school_year_start_month, school_year_start_day, updated_at
) VALUES ($1, $2, $3, $4, now())
ON CONFLICT (parent_user_id) DO UPDATE SET
    school_name = EXCLUDED.school_name,
    school_year_start_month = EXCLUDED.school_year_start_month,
    school_year_start_day = EXCLUDED.school_year_start_day,
    updated_at = now()
RETURNING parent_user_id, school_name, school_year_start_month, school_year_start_day, updated_at
`, parentID, schoolName, startMonth, startDay).Scan(
		&s.ParentUserID, &s.SchoolName, &s.SchoolYearStartMonth, &s.SchoolYearStartDay, &s.UpdatedAt,
	)
	return s, err
}
