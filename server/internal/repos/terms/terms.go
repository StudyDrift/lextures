// Package terms provides org-scoped academic terms (plan 5.3).
package terms

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTermNotFound      = errors.New("terms: not found")
	ErrTermWrongOrg      = errors.New("terms: wrong organization")
	ErrTermHasCourses    = errors.New("terms: cannot delete term with courses")
	ErrTermDuplicate     = errors.New("terms: a term with this name already exists")
	ErrInvalidTermType   = errors.New("terms: invalid term_type")
	ErrInvalidTermStatus = errors.New("terms: invalid status")
	ErrInvalidDateRange  = errors.New("terms: end_date must be after start_date")
)

func isUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

// TermPublic is JSON for APIs (camelCase).
type TermPublic struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"orgId"`
	Name      string    `json:"name"`
	TermType  string    `json:"termType"`
	StartDate string    `json:"startDate"` // YYYY-MM-DD
	EndDate   string    `json:"endDate"`   // YYYY-MM-DD
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func validateTermType(t string) bool {
	switch strings.TrimSpace(strings.ToLower(t)) {
	case "semester", "quarter", "trimester", "year", "grading_period", "custom":
		return true
	default:
		return false
	}
}

func validateStatus(s string) bool {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "upcoming", "active", "completed", "archived":
		return true
	default:
		return false
	}
}

func scanTerm(row pgx.Row) (TermPublic, error) {
	var (
		id, orgID                            uuid.UUID
		name, termType, startD, endD, status string
		createdAt, updatedAt                 time.Time
	)
	if err := row.Scan(&id, &orgID, &name, &termType, &startD, &endD, &status, &createdAt, &updatedAt); err != nil {
		return TermPublic{}, err
	}
	return TermPublic{
		ID:        id.String(),
		OrgID:     orgID.String(),
		Name:      name,
		TermType:  termType,
		StartDate: startD,
		EndDate:   endD,
		Status:    status,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// ListByOrg returns terms for an organization ordered by start_date descending.
func ListByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]TermPublic, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, name, term_type, start_date::text, end_date::text, status, created_at, updated_at
FROM tenant.terms
WHERE org_id = $1
ORDER BY start_date DESC, name ASC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TermPublic
	for rows.Next() {
		t, err := scanTerm(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetByID loads a term by id or returns nil.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*TermPublic, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, name, term_type, start_date::text, end_date::text, status, created_at, updated_at
FROM tenant.terms WHERE id = $1
`, id)
	t, err := scanTerm(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FindByOrgName returns the term for an org with a case-insensitive name match, or nil.
func FindByOrgName(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, name string) (*TermPublic, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
SELECT id, org_id, name, term_type, start_date::text, end_date::text, status, created_at, updated_at
FROM tenant.terms
WHERE org_id = $1 AND lower(btrim(name)) = lower(btrim($2::text))
ORDER BY start_date DESC, created_at ASC
LIMIT 1
`, orgID, name)
	t, err := scanTerm(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// EquivalentIDs returns all term IDs in the same org that share the selected term's name
// (case-insensitive). Used so course filters still work if duplicate rows remain.
func EquivalentIDs(ctx context.Context, pool *pgxpool.Pool, termID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT t2.id
FROM tenant.terms t1
INNER JOIN tenant.terms t2
  ON t2.org_id = t1.org_id
 AND lower(btrim(t2.name)) = lower(btrim(t1.name))
WHERE t1.id = $1
`, termID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []uuid.UUID{termID}, nil
	}
	return out, nil
}

// Create inserts a term; status defaults to DeriveStatusFromDates when empty.
// Returns ErrTermDuplicate when the org already has a term with the same name (case-insensitive).
func Create(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, name, termType, startDate, endDate, status string) (*TermPublic, error) {
	name = strings.TrimSpace(name)
	termType = strings.TrimSpace(strings.ToLower(termType))
	if termType == "" {
		termType = "semester"
	}
	if !validateTermType(termType) {
		return nil, ErrInvalidTermType
	}
	if name == "" {
		return nil, errors.New("terms: name required")
	}
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	if startDate == "" || endDate == "" {
		return nil, errors.New("terms: start_date and end_date required")
	}
	if startDate >= endDate {
		return nil, ErrInvalidDateRange
	}
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		status = DeriveStatusFromDates(time.Now().UTC(), startDate, endDate)
	}
	if !validateStatus(status) {
		return nil, ErrInvalidTermStatus
	}
	existing, err := FindByOrgName(ctx, pool, orgID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrTermDuplicate
	}
	row := pool.QueryRow(ctx, `
INSERT INTO tenant.terms (org_id, name, term_type, start_date, end_date, status)
VALUES ($1, $2, $3, $4::date, $5::date, $6)
RETURNING id, org_id, name, term_type, start_date::text, end_date::text, status, created_at, updated_at
`, orgID, name, termType, startDate, endDate, status)
	t, err := scanTerm(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTermDuplicate
		}
		return nil, err
	}
	return &t, nil
}

// Ensure returns an existing term with the same org+name, or creates one.
// When an existing term is found, start/end/status from the request are ignored.
func Ensure(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, name, termType, startDate, endDate, status string) (*TermPublic, error) {
	existing, err := FindByOrgName(ctx, pool, orgID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	t, err := Create(ctx, pool, orgID, name, termType, startDate, endDate, status)
	if err == nil {
		return t, nil
	}
	if errors.Is(err, ErrTermDuplicate) {
		return FindByOrgName(ctx, pool, orgID, name)
	}
	return nil, err
}

// Patch updates mutable fields; nil means omit.
func Patch(ctx context.Context, pool *pgxpool.Pool, orgID, termID uuid.UUID, name *string, termType *string, startDate *string, endDate *string, status *string) (*TermPublic, error) {
	cur, err := GetByID(ctx, pool, termID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrTermNotFound
	}
	if cur.OrgID != orgID.String() {
		return nil, ErrTermWrongOrg
	}
	n := cur.Name
	tt := cur.TermType
	sd := cur.StartDate
	ed := cur.EndDate
	st := cur.Status
	if name != nil {
		n = strings.TrimSpace(*name)
		if n == "" {
			return nil, errors.New("terms: name required")
		}
	}
	if termType != nil {
		tt = strings.TrimSpace(strings.ToLower(*termType))
		if !validateTermType(tt) {
			return nil, ErrInvalidTermType
		}
	}
	if startDate != nil {
		sd = strings.TrimSpace(*startDate)
	}
	if endDate != nil {
		ed = strings.TrimSpace(*endDate)
	}
	if sd >= ed {
		return nil, ErrInvalidDateRange
	}
	if status != nil {
		st = strings.TrimSpace(strings.ToLower(*status))
		if !validateStatus(st) {
			return nil, ErrInvalidTermStatus
		}
	}
	if !strings.EqualFold(strings.TrimSpace(n), strings.TrimSpace(cur.Name)) {
		clash, err := FindByOrgName(ctx, pool, orgID, n)
		if err != nil {
			return nil, err
		}
		if clash != nil && clash.ID != termID.String() {
			return nil, ErrTermDuplicate
		}
	}
	row := pool.QueryRow(ctx, `
UPDATE tenant.terms
SET name = $2, term_type = $3, start_date = $4::date, end_date = $5::date, status = $6, updated_at = NOW()
WHERE id = $1 AND org_id = $7
RETURNING id, org_id, name, term_type, start_date::text, end_date::text, status, created_at, updated_at
`, termID, n, tt, sd, ed, st, orgID)
	t, err := scanTerm(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTermNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTermDuplicate
		}
		return nil, err
	}
	return &t, nil
}

// Delete removes a term only when no courses reference it.
func Delete(ctx context.Context, pool *pgxpool.Pool, orgID, termID uuid.UUID) error {
	cur, err := GetByID(ctx, pool, termID)
	if err != nil {
		return err
	}
	if cur == nil {
		return ErrTermNotFound
	}
	if cur.OrgID != orgID.String() {
		return ErrTermWrongOrg
	}
	var n int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.courses WHERE term_id = $1`, termID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrTermHasCourses
	}
	tag, err := pool.Exec(ctx, `DELETE FROM tenant.terms WHERE id = $1 AND org_id = $2`, termID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTermNotFound
	}
	return nil
}

// DeriveStatusFromDates picks upcoming/active/completed from wall-clock date in UTC.
func DeriveStatusFromDates(now time.Time, startDateStr, endDateStr string) string {
	start, err1 := time.ParseInLocation("2006-01-02", startDateStr, time.UTC)
	end, err2 := time.ParseInLocation("2006-01-02", endDateStr, time.UTC)
	if err1 != nil || err2 != nil {
		return "upcoming"
	}
	today := now.UTC().Truncate(24 * time.Hour)
	startDay := start.UTC().Truncate(24 * time.Hour)
	endDay := end.UTC().Truncate(24 * time.Hour)
	if today.Before(startDay) {
		return "upcoming"
	}
	if today.After(endDay) {
		return "completed"
	}
	return "active"
}

// SweepStatuses transitions upcoming/active/completed based on calendar dates (UTC). Idempotent.
func SweepStatuses(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int64, error) {
	today := now.UTC().Format("2006-01-02")
	tag, err := pool.Exec(ctx, `
UPDATE tenant.terms t
SET status = CASE
    WHEN $1::date < t.start_date THEN 'upcoming'
    WHEN $1::date > t.end_date THEN 'completed'
    ELSE 'active'
END,
updated_at = NOW()
WHERE t.status <> 'archived'
  AND (
    ($1::date < t.start_date AND t.status <> 'upcoming')
    OR ($1::date > t.end_date AND t.status <> 'completed')
    OR ($1::date >= t.start_date AND $1::date <= t.end_date AND t.status NOT IN ('active', 'archived'))
  )
`, today)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
