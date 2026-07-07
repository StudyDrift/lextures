package reportcards

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DateRange is an inclusive UTC calendar range for attendance/report-card scoping.
type DateRange struct {
	Start time.Time
	End   time.Time
}

var (
	quarterPeriodRe = regexp.MustCompile(`(?i)^Q([1-4])-(\d{4})$`)
	semesterPeriodRe = regexp.MustCompile(`(?i)^S([12])-(\d{4})$`)
)

// ResolveGradingPeriodDateRange maps a grading-period label (e.g. Q1-2026) to calendar dates.
// It prefers an org term whose name matches the label, then falls back to Q/S pattern parsing.
func ResolveGradingPeriodDateRange(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, gradingPeriod string) (DateRange, bool, error) {
	gradingPeriod = strings.TrimSpace(gradingPeriod)
	if gradingPeriod == "" {
		return DateRange{}, false, nil
	}

	var startStr, endStr string
	err := pool.QueryRow(ctx, `
SELECT start_date::text, end_date::text
FROM tenant.terms
WHERE org_id = $1 AND lower(trim(name)) = lower(trim($2))
ORDER BY start_date DESC
LIMIT 1
`, orgID, gradingPeriod).Scan(&startStr, &endStr)
	if err == nil {
		return dateRangeFromStrings(startStr, endStr)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DateRange{}, false, err
	}

	if dr, ok := parseGradingPeriodLabel(gradingPeriod); ok {
		return dr, true, nil
	}
	return DateRange{}, false, nil
}

func dateRangeFromStrings(startStr, endStr string) (DateRange, bool, error) {
	start, err := time.Parse("2006-01-02", strings.TrimSpace(startStr))
	if err != nil {
		return DateRange{}, false, err
	}
	end, err := time.Parse("2006-01-02", strings.TrimSpace(endStr))
	if err != nil {
		return DateRange{}, false, err
	}
	return DateRange{
		Start: start.UTC().Truncate(24 * time.Hour),
		End:   end.UTC().Truncate(24 * time.Hour),
	}, true, nil
}

func parseGradingPeriodLabel(label string) (DateRange, bool) {
	label = strings.TrimSpace(label)
	if m := quarterPeriodRe.FindStringSubmatch(label); m != nil {
		q, _ := strconv.Atoi(m[1])
		year, _ := strconv.Atoi(m[2])
		return quarterDateRange(q, year), true
	}
	if m := semesterPeriodRe.FindStringSubmatch(label); m != nil {
		half, _ := strconv.Atoi(m[1])
		year, _ := strconv.Atoi(m[2])
		return semesterDateRange(half, year), true
	}
	return DateRange{}, false
}

func quarterDateRange(quarter, year int) DateRange {
	switch quarter {
	case 1:
		return calendarRange(year, time.January, 1, year, time.March, 31)
	case 2:
		return calendarRange(year, time.April, 1, year, time.June, 30)
	case 3:
		return calendarRange(year, time.July, 1, year, time.September, 30)
	default:
		return calendarRange(year, time.October, 1, year, time.December, 31)
	}
}

func semesterDateRange(half, year int) DateRange {
	if half == 1 {
		return calendarRange(year, time.January, 1, year, time.June, 30)
	}
	return calendarRange(year, time.July, 1, year, time.December, 31)
}

func calendarRange(startYear int, startMonth time.Month, startDay, endYear int, endMonth time.Month, endDay int) DateRange {
	return DateRange{
		Start: time.Date(startYear, startMonth, startDay, 0, 0, 0, 0, time.UTC),
		End:   time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, time.UTC),
	}
}