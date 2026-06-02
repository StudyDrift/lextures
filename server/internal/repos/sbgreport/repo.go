// Package sbgreport persists plan 13.5 standards-based grading data (domains,
// standards, mastery scales, mastery scores).
package sbgreport

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Domain types ─────────────────────────────────────────────────────────────

// StandardDomain is one row from sbg.standard_domains.
type StandardDomain struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	Code       string
	Name       string
	GradeLevel *string
	CreatedAt  time.Time
}

// Standard is one row from sbg.standards.
type Standard struct {
	ID          uuid.UUID
	DomainID    uuid.UUID
	Code        string
	Description string
	CreatedAt   time.Time
}

// MasteryScale is one row from sbg.mastery_scales.
type MasteryScale struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Label     string
	Value     int
	Color     *string
	CreatedAt time.Time
}

// MasteryScore is one row from sbg.mastery_scores.
type MasteryScore struct {
	ID            uuid.UUID
	StudentID     uuid.UUID
	StandardID    uuid.UUID
	CourseID      uuid.UUID
	GradingPeriod string
	ScoreValue    int
	AssessedBy    *uuid.UUID
	Source        string
	SourceID      *uuid.UUID
	AssessedAt    time.Time
}

// HeatmapCell is one student×standard cell in the mastery heatmap.
type HeatmapCell struct {
	StudentID  uuid.UUID
	StandardID uuid.UUID
	ScoreValue int // most-recent score
}

// ─── Standard Domains ─────────────────────────────────────────────────────────

// ListStandardDomains returns all domains for an org, with their standards.
func ListStandardDomains(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]StandardDomain, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, code, name, grade_level, created_at
FROM sbg.standard_domains
WHERE org_id = $1
ORDER BY grade_level NULLS LAST, code`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StandardDomain
	for rows.Next() {
		var d StandardDomain
		if err := rows.Scan(&d.ID, &d.OrgID, &d.Code, &d.Name, &d.GradeLevel, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CreateStandardDomain inserts a new domain. Returns the created row.
func CreateStandardDomain(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, code, name string, gradeLevel *string) (*StandardDomain, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO sbg.standard_domains (org_id, code, name, grade_level)
VALUES ($1, $2, $3, $4)
ON CONFLICT (org_id, code) DO UPDATE SET name = EXCLUDED.name, grade_level = EXCLUDED.grade_level
RETURNING id, org_id, code, name, grade_level, created_at`, orgID, code, name, gradeLevel)
	var d StandardDomain
	if err := row.Scan(&d.ID, &d.OrgID, &d.Code, &d.Name, &d.GradeLevel, &d.CreatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

// ─── Standards ────────────────────────────────────────────────────────────────

// ListStandardsForDomain returns all standards belonging to a domain.
func ListStandardsForDomain(ctx context.Context, pool *pgxpool.Pool, domainID uuid.UUID) ([]Standard, error) {
	rows, err := pool.Query(ctx, `
SELECT id, domain_id, code, description, created_at
FROM sbg.standards
WHERE domain_id = $1
ORDER BY code`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Standard
	for rows.Next() {
		var s Standard
		if err := rows.Scan(&s.ID, &s.DomainID, &s.Code, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListStandardsForOrg returns all standards for all domains of an org, joined with domain info.
// Returns (standards, domainByStandardID) — simple flat list used by course views.
func ListStandardsForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Standard, error) {
	rows, err := pool.Query(ctx, `
SELECT s.id, s.domain_id, s.code, s.description, s.created_at
FROM sbg.standards s
JOIN sbg.standard_domains d ON d.id = s.domain_id
WHERE d.org_id = $1
ORDER BY d.grade_level NULLS LAST, d.code, s.code`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Standard
	for rows.Next() {
		var s Standard
		if err := rows.Scan(&s.ID, &s.DomainID, &s.Code, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpsertStandard inserts or updates a standard (keyed by domain + code).
func UpsertStandard(ctx context.Context, pool *pgxpool.Pool, domainID uuid.UUID, code, description string) (*Standard, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO sbg.standards (domain_id, code, description)
VALUES ($1, $2, $3)
ON CONFLICT (domain_id, code) DO UPDATE SET description = EXCLUDED.description
RETURNING id, domain_id, code, description, created_at`, domainID, code, description)
	var s Standard
	if err := row.Scan(&s.ID, &s.DomainID, &s.Code, &s.Description, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

// CSVImportResult summarises a CSV standards import.
type CSVImportResult struct {
	DomainsCreated   int
	StandardsImported int
	Errors           []string
}

// ImportStandardsCSV reads a CSV (code, description, domain_code, domain_name, grade_level) and
// bulk-upserts standard domains + standards for the org. The CSV must have a header row.
func ImportStandardsCSV(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, r io.Reader) (*CSVImportResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	required := []string{"code", "description", "domain_code", "domain_name"}
	for _, req := range required {
		if _, ok := colIdx[req]; !ok {
			return nil, fmt.Errorf("missing required CSV column: %s", req)
		}
	}

	result := &CSVImportResult{}
	domainCache := map[string]*StandardDomain{}

	for lineNum := 2; ; lineNum++ {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: %v", lineNum, err))
			continue
		}

		get := func(col string) string {
			i, ok := colIdx[col]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}

		code := get("code")
		description := get("description")
		domainCode := get("domain_code")
		domainName := get("domain_name")
		gradeLevel := get("grade_level")

		if code == "" || description == "" || domainCode == "" || domainName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: required fields missing", lineNum))
			continue
		}

		domain, ok := domainCache[domainCode]
		if !ok {
			var gl *string
			if gradeLevel != "" {
				gl = &gradeLevel
			}
			d, err := CreateStandardDomain(ctx, pool, orgID, domainCode, domainName, gl)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("line %d domain %s: %v", lineNum, domainCode, err))
				continue
			}
			domainCache[domainCode] = d
			domain = d
			result.DomainsCreated++
		}

		if _, err := UpsertStandard(ctx, pool, domain.ID, code, description); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d standard %s: %v", lineNum, code, err))
			continue
		}
		result.StandardsImported++
	}
	return result, nil
}

// ─── Mastery Scales ───────────────────────────────────────────────────────────

// ListMasteryScales returns the ordered mastery scale for an org.
func ListMasteryScales(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]MasteryScale, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, label, value, color, created_at
FROM sbg.mastery_scales
WHERE org_id = $1
ORDER BY value DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MasteryScale
	for rows.Next() {
		var m MasteryScale
		if err := rows.Scan(&m.ID, &m.OrgID, &m.Label, &m.Value, &m.Color, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ReplaceMasteryScale replaces all mastery scale levels for an org atomically.
type MasteryScaleEntry struct {
	Label string
	Value int
	Color *string
}

func ReplaceMasteryScale(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, entries []MasteryScaleEntry) ([]MasteryScale, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `DELETE FROM sbg.mastery_scales WHERE org_id = $1`, orgID); err != nil {
		return nil, err
	}
	var out []MasteryScale
	for _, e := range entries {
		var m MasteryScale
		row := tx.QueryRow(ctx, `
INSERT INTO sbg.mastery_scales (org_id, label, value, color)
VALUES ($1, $2, $3, $4)
RETURNING id, org_id, label, value, color, created_at`, orgID, e.Label, e.Value, e.Color)
		if err := row.Scan(&m.ID, &m.OrgID, &m.Label, &m.Value, &m.Color, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, tx.Commit(ctx)
}

// SeedDefaultMasteryScale inserts the default 4-level scale if no scale exists for the org.
func SeedDefaultMasteryScale(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]MasteryScale, error) {
	existing, err := ListMasteryScales(ctx, pool, orgID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return existing, nil
	}
	defaults := []MasteryScaleEntry{
		{Label: "Exceeds Standard", Value: 4, Color: stringPtr("#22c55e")},
		{Label: "Meets Standard", Value: 3, Color: stringPtr("#3b82f6")},
		{Label: "Approaching Standard", Value: 2, Color: stringPtr("#f59e0b")},
		{Label: "Below Standard", Value: 1, Color: stringPtr("#ef4444")},
	}
	return ReplaceMasteryScale(ctx, pool, orgID, defaults)
}

func stringPtr(s string) *string { return &s }

// ─── Mastery Scores ───────────────────────────────────────────────────────────

// RecordMasteryScore inserts a new mastery score evidence record.
func RecordMasteryScore(ctx context.Context, pool *pgxpool.Pool, studentID, standardID, courseID uuid.UUID, period string, scoreValue int, assessedBy *uuid.UUID, source string, sourceID *uuid.UUID) (*MasteryScore, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO sbg.mastery_scores (student_id, standard_id, course_id, grading_period, score_value, assessed_by, source, source_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, student_id, standard_id, course_id, grading_period, score_value, assessed_by, source, source_id, assessed_at`,
		studentID, standardID, courseID, period, scoreValue, assessedBy, source, sourceID)
	return scanMasteryScore(row)
}

// ListMasteryScoresForStudentPeriod returns all evidence records for a student in a course+period.
func ListMasteryScoresForStudentPeriod(ctx context.Context, pool *pgxpool.Pool, studentID, courseID uuid.UUID, period string) ([]MasteryScore, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, standard_id, course_id, grading_period, score_value, assessed_by, source, source_id, assessed_at
FROM sbg.mastery_scores
WHERE student_id = $1 AND course_id = $2 AND grading_period = $3
ORDER BY standard_id, assessed_at`, studentID, courseID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasteryScoreRows(rows)
}

// ListMasteryScoresForCoursePeriod returns all evidence records for a course+period (all students).
func ListMasteryScoresForCoursePeriod(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, period string) ([]MasteryScore, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, standard_id, course_id, grading_period, score_value, assessed_by, source, source_id, assessed_at
FROM sbg.mastery_scores
WHERE course_id = $1 AND grading_period = $2
ORDER BY student_id, standard_id, assessed_at`, courseID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMasteryScoreRows(rows)
}

// GetHeatmap returns the most-recent score per (student, standard) for a course+period.
func GetHeatmap(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, period string) ([]HeatmapCell, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (student_id, standard_id) student_id, standard_id, score_value
FROM sbg.mastery_scores
WHERE course_id = $1 AND grading_period = $2
ORDER BY student_id, standard_id, assessed_at DESC`, courseID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HeatmapCell
	for rows.Next() {
		var c HeatmapCell
		if err := rows.Scan(&c.StudentID, &c.StandardID, &c.ScoreValue); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetStandardByID returns a single standard or nil if not found.
func GetStandardByID(ctx context.Context, pool *pgxpool.Pool, standardID uuid.UUID) (*Standard, error) {
	row := pool.QueryRow(ctx, `
SELECT id, domain_id, code, description, created_at FROM sbg.standards WHERE id = $1`, standardID)
	var s Standard
	if err := row.Scan(&s.ID, &s.DomainID, &s.Code, &s.Description, &s.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// GetDomainOrgID returns the org_id for the domain that owns a standard.
func GetDomainOrgID(ctx context.Context, pool *pgxpool.Pool, standardID uuid.UUID) (*uuid.UUID, error) {
	var orgID uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT d.org_id FROM sbg.standard_domains d
JOIN sbg.standards s ON s.domain_id = d.id
WHERE s.id = $1`, standardID).Scan(&orgID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &orgID, err
}

// ─── Scan helpers ─────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMasteryScore(row rowScanner) (*MasteryScore, error) {
	var m MasteryScore
	if err := row.Scan(
		&m.ID, &m.StudentID, &m.StandardID, &m.CourseID, &m.GradingPeriod,
		&m.ScoreValue, &m.AssessedBy, &m.Source, &m.SourceID, &m.AssessedAt,
	); err != nil {
		return nil, err
	}
	return &m, nil
}

func scanMasteryScoreRows(rows pgx.Rows) ([]MasteryScore, error) {
	var out []MasteryScore
	for rows.Next() {
		var m MasteryScore
		if err := rows.Scan(
			&m.ID, &m.StudentID, &m.StandardID, &m.CourseID, &m.GradingPeriod,
			&m.ScoreValue, &m.AssessedBy, &m.Source, &m.SourceID, &m.AssessedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
