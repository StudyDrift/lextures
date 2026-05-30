// Package behavior persists plan 13.3 PBIS behavior tracking records.
package behavior

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// errCategoryInUse is returned when a category is referenced by existing records.
var errCategoryInUse = errors.New("category is referenced by existing records")

// Category is one row from behavior.categories.
type Category struct {
	ID     uuid.UUID
	OrgID  uuid.UUID
	Name   string
	Type   string // positive | negative
	Color  *string
	Active bool
}

// Award is one row from behavior.pbis_awards.
type Award struct {
	ID           uuid.UUID
	StudentID    uuid.UUID
	AwardedBy    uuid.UUID
	CategoryID   uuid.UUID
	CategoryName string
	OrgID        uuid.UUID
	Points       int
	Note         *string
	AwardedAt    time.Time
}

// Referral is one row from behavior.referrals.
type Referral struct {
	ID           uuid.UUID
	StudentID    uuid.UUID
	FiledBy      uuid.UUID
	OrgID        uuid.UUID
	SchoolID     *uuid.UUID
	CategoryID   uuid.UUID
	CategoryName string
	IncidentAt   time.Time
	Location     *string
	Description  string
	Response     *string
	CreatedAt    time.Time
}

// AwardInput is the input for creating a PBIS award.
type AwardInput struct {
	StudentID  uuid.UUID
	AwardedBy  uuid.UUID
	CategoryID uuid.UUID
	OrgID      uuid.UUID
	Points     int
	Note       *string
}

// ReferralInput is the input for filing a behavior referral.
type ReferralInput struct {
	StudentID   uuid.UUID
	FiledBy     uuid.UUID
	OrgID       uuid.UUID
	SchoolID    *uuid.UUID
	CategoryID  uuid.UUID
	IncidentAt  time.Time
	Location    *string
	Description string
	Response    *string
}

// StudentSummary aggregates behavior data for one student.
type StudentSummary struct {
	StudentID    uuid.UUID
	TotalPoints  int
	PointsByCategory []CategoryPoints
	Referrals    []ReferralSummary
}

// CategoryPoints is a point total for one category.
type CategoryPoints struct {
	CategoryID   uuid.UUID
	CategoryName string
	Points       int
}

// ReferralSummary is a limited view of a referral for parents.
type ReferralSummary struct {
	ID           uuid.UUID
	CategoryName string
	IncidentAt   time.Time
}

// DashboardData holds school-level PBIS analytics.
type DashboardData struct {
	WeekStart       time.Time
	TotalPoints     int
	TotalReferrals  int
	PointsByCategory []CategoryPoints
	ReferralsByCategory []CategoryReferrals
}

// CategoryReferrals is a referral count for one category.
type CategoryReferrals struct {
	CategoryID   uuid.UUID
	CategoryName string
	Count        int
}

// ListCategories returns all categories for an org ordered by name.
func ListCategories(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Category, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, name, type, color, active
FROM behavior.categories
WHERE org_id = $1
ORDER BY type ASC, name ASC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		var color sql.NullString
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.Type, &color, &c.Active); err != nil {
			return nil, err
		}
		if color.Valid {
			c.Color = &color.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpsertCategory creates or updates a behavior category.
func UpsertCategory(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, name, catType string, color *string) (*Category, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO behavior.categories (org_id, name, type, color)
VALUES ($1, $2, $3, $4)
ON CONFLICT (org_id, name) DO UPDATE
    SET type = EXCLUDED.type,
        color = EXCLUDED.color,
        active = true
RETURNING id, org_id, name, type, color, active
`, orgID, name, catType, color)
	var c Category
	var col sql.NullString
	if err := row.Scan(&c.ID, &c.OrgID, &c.Name, &c.Type, &col, &c.Active); err != nil {
		return nil, err
	}
	if col.Valid {
		c.Color = &col.String
	}
	return &c, nil
}

// DeleteCategory soft-deletes a category (sets active=false) if no records reference it.
// Returns (true, nil) if deactivated, (false, nil) if not found, (false, err) on conflict.
func DeleteCategory(ctx context.Context, pool *pgxpool.Pool, orgID, categoryID uuid.UUID) (bool, error) {
	var inUseAward, inUseReferral bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM behavior.pbis_awards WHERE category_id = $1)`,
		categoryID).Scan(&inUseAward); err != nil {
		return false, err
	}
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM behavior.referrals WHERE category_id = $1)`,
		categoryID).Scan(&inUseReferral); err != nil {
		return false, err
	}
	if inUseAward || inUseReferral {
		return false, errCategoryInUse
	}
	tag, err := pool.Exec(ctx,
		`UPDATE behavior.categories SET active = false WHERE id = $1 AND org_id = $2`,
		categoryID, orgID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SeedDefaultCategories inserts the default PBIS categories for an org if none exist.
func SeedDefaultCategories(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) error {
	defaults := []struct {
		name, catType, color string
	}{
		{"Respect", "positive", "#4CAF50"},
		{"Responsibility", "positive", "#2196F3"},
		{"Safety", "positive", "#FF9800"},
		{"Kindness", "positive", "#E91E63"},
		{"Disruptive Behavior", "negative", "#F44336"},
		{"Disrespect", "negative", "#9C27B0"},
	}
	for _, d := range defaults {
		col := d.color
		_, err := pool.Exec(ctx, `
INSERT INTO behavior.categories (org_id, name, type, color)
VALUES ($1, $2, $3, $4)
ON CONFLICT (org_id, name) DO NOTHING
`, orgID, d.name, d.catType, col)
		if err != nil {
			return err
		}
	}
	return nil
}

// BatchAwardPoints creates multiple PBIS awards in a single transaction.
func BatchAwardPoints(ctx context.Context, pool *pgxpool.Pool, inputs []AwardInput) ([]Award, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var out []Award
	for _, inp := range inputs {
		points := inp.Points
		if points <= 0 {
			points = 1
		}
		var a Award
		var catName string
		var note sql.NullString
		err := tx.QueryRow(ctx, `
WITH inserted AS (
    INSERT INTO behavior.pbis_awards (student_id, awarded_by, category_id, org_id, points, note)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, student_id, awarded_by, category_id, org_id, points, note, awarded_at
)
SELECT i.id, i.student_id, i.awarded_by, i.category_id, i.org_id, i.points, i.note, i.awarded_at, bc.name
FROM inserted i
JOIN behavior.categories bc ON bc.id = i.category_id
`,
			inp.StudentID, inp.AwardedBy, inp.CategoryID, inp.OrgID, points, inp.Note,
		).Scan(&a.ID, &a.StudentID, &a.AwardedBy, &a.CategoryID, &a.OrgID, &a.Points, &note, &a.AwardedAt, &catName)
		if err != nil {
			return nil, err
		}
		if note.Valid {
			a.Note = &note.String
		}
		a.CategoryName = catName
		out = append(out, a)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// FileReferral creates a new behavior referral.
func FileReferral(ctx context.Context, pool *pgxpool.Pool, inp ReferralInput) (*Referral, error) {
	var r Referral
	var catName string
	var schoolID sql.NullString
	var location, response sql.NullString
	err := pool.QueryRow(ctx, `
WITH inserted AS (
    INSERT INTO behavior.referrals
        (student_id, filed_by, org_id, school_id, category_id, incident_at, location, description, response)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id, student_id, filed_by, org_id, school_id, category_id, incident_at, location, description, response, created_at
)
SELECT i.id, i.student_id, i.filed_by, i.org_id, i.school_id, i.category_id,
       i.incident_at, i.location, i.description, i.response, i.created_at, bc.name
FROM inserted i
JOIN behavior.categories bc ON bc.id = i.category_id
`,
		inp.StudentID, inp.FiledBy, inp.OrgID, inp.SchoolID, inp.CategoryID,
		inp.IncidentAt, inp.Location, inp.Description, inp.Response,
	).Scan(
		&r.ID, &r.StudentID, &r.FiledBy, &r.OrgID, &schoolID, &r.CategoryID,
		&r.IncidentAt, &location, &r.Description, &response, &r.CreatedAt, &catName,
	)
	if err != nil {
		return nil, err
	}
	if schoolID.Valid && schoolID.String != "" {
		u, err := uuid.Parse(schoolID.String)
		if err != nil {
			return nil, err
		}
		r.SchoolID = &u
	}
	if location.Valid {
		r.Location = &location.String
	}
	if response.Valid {
		r.Response = &response.String
	}
	r.CategoryName = catName
	return &r, nil
}

// ListAwardsForStudent returns PBIS awards for a student, newest first.
func ListAwardsForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, limit int) ([]Award, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := pool.Query(ctx, `
SELECT pa.id, pa.student_id, pa.awarded_by, pa.category_id, pa.org_id, pa.points, pa.note, pa.awarded_at, bc.name
FROM behavior.pbis_awards pa
JOIN behavior.categories bc ON bc.id = pa.category_id
WHERE pa.student_id = $1
ORDER BY pa.awarded_at DESC
LIMIT $2
`, studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAwards(rows)
}

// ListReferralsForStudent returns referrals for a student, newest first.
// fullDescription controls whether the description field is returned (FERPA: admin/teacher only).
func ListReferralsForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, limit int, fullDescription bool) ([]Referral, error) {
	if limit <= 0 {
		limit = 200
	}
	descExpr := `''` // parents see no description
	if fullDescription {
		descExpr = `r.description`
	}
	query := `
SELECT r.id, r.student_id, r.filed_by, r.org_id, r.school_id, r.category_id,
       r.incident_at, r.location, ` + descExpr + `, r.response, r.created_at, bc.name
FROM behavior.referrals r
JOIN behavior.categories bc ON bc.id = r.category_id
WHERE r.student_id = $1
ORDER BY r.incident_at DESC
LIMIT $2`
	rows, err := pool.Query(ctx, query, studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReferrals(rows)
}

// OrgDashboard returns school-level PBIS analytics for the current week.
func OrgDashboard(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, weekStart time.Time) (*DashboardData, error) {
	weekEnd := weekStart.AddDate(0, 0, 7)

	var totalPoints int
	if err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(points), 0)
FROM behavior.pbis_awards
WHERE org_id = $1 AND awarded_at >= $2 AND awarded_at < $3
`, orgID, weekStart, weekEnd).Scan(&totalPoints); err != nil {
		return nil, err
	}

	var totalReferrals int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM behavior.referrals
WHERE org_id = $1 AND incident_at >= $2 AND incident_at < $3
`, orgID, weekStart, weekEnd).Scan(&totalReferrals); err != nil {
		return nil, err
	}

	pointsRows, err := pool.Query(ctx, `
SELECT pa.category_id, bc.name, SUM(pa.points)
FROM behavior.pbis_awards pa
JOIN behavior.categories bc ON bc.id = pa.category_id
WHERE pa.org_id = $1 AND pa.awarded_at >= $2 AND pa.awarded_at < $3
GROUP BY pa.category_id, bc.name
ORDER BY SUM(pa.points) DESC
LIMIT 10
`, orgID, weekStart, weekEnd)
	if err != nil {
		return nil, err
	}
	defer pointsRows.Close()
	var pointsByCat []CategoryPoints
	for pointsRows.Next() {
		var cp CategoryPoints
		if err := pointsRows.Scan(&cp.CategoryID, &cp.CategoryName, &cp.Points); err != nil {
			return nil, err
		}
		pointsByCat = append(pointsByCat, cp)
	}
	if err := pointsRows.Err(); err != nil {
		return nil, err
	}

	refRows, err := pool.Query(ctx, `
SELECT r.category_id, bc.name, COUNT(*)
FROM behavior.referrals r
JOIN behavior.categories bc ON bc.id = r.category_id
WHERE r.org_id = $1 AND r.incident_at >= $2 AND r.incident_at < $3
GROUP BY r.category_id, bc.name
ORDER BY COUNT(*) DESC
LIMIT 10
`, orgID, weekStart, weekEnd)
	if err != nil {
		return nil, err
	}
	defer refRows.Close()
	var refsByCat []CategoryReferrals
	for refRows.Next() {
		var cr CategoryReferrals
		if err := refRows.Scan(&cr.CategoryID, &cr.CategoryName, &cr.Count); err != nil {
			return nil, err
		}
		refsByCat = append(refsByCat, cr)
	}
	if err := refRows.Err(); err != nil {
		return nil, err
	}

	return &DashboardData{
		WeekStart:           weekStart,
		TotalPoints:         totalPoints,
		TotalReferrals:      totalReferrals,
		PointsByCategory:    pointsByCat,
		ReferralsByCategory: refsByCat,
	}, nil
}

func scanAwards(rows pgx.Rows) ([]Award, error) {
	var out []Award
	for rows.Next() {
		var a Award
		var note sql.NullString
		if err := rows.Scan(
			&a.ID, &a.StudentID, &a.AwardedBy, &a.CategoryID, &a.OrgID,
			&a.Points, &note, &a.AwardedAt, &a.CategoryName,
		); err != nil {
			return nil, err
		}
		if note.Valid {
			a.Note = &note.String
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanReferrals(rows pgx.Rows) ([]Referral, error) {
	var out []Referral
	for rows.Next() {
		var r Referral
		var schoolID, location, response sql.NullString
		if err := rows.Scan(
			&r.ID, &r.StudentID, &r.FiledBy, &r.OrgID, &schoolID, &r.CategoryID,
			&r.IncidentAt, &location, &r.Description, &response, &r.CreatedAt, &r.CategoryName,
		); err != nil {
			return nil, err
		}
		if schoolID.Valid && schoolID.String != "" {
			u, err := uuid.Parse(schoolID.String)
			if err != nil {
				return nil, err
			}
			r.SchoolID = &u
		}
		if location.Valid {
			r.Location = &location.String
		}
		if response.Valid {
			r.Response = &response.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
