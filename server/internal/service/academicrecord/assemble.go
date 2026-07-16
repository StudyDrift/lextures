package academicrecord

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AssembleParams controls academic-record assembly.
type AssembleParams struct {
	UserID          uuid.UUID
	Variant         Variant
	TermIDs         []uuid.UUID // optional filter for partial transcripts
	InstitutionName string
	StudentName     string
	StudentID       string
	Scale           ScaleKind
	GeneratedAt     time.Time
}

// CourseRow is one enrollment line loaded from the database.
type CourseRow struct {
	CourseID         uuid.UUID
	EnrollmentID     uuid.UUID
	CourseCode       string
	Title            string
	Credits          float64
	FinalGrade       *string
	TermID           *uuid.UUID
	TermName         *string
	TermStart        *time.Time
	TermEnd          *time.Time
	EnrollmentState  string
}

// LoadCourseRows loads graded and in-progress enrollments for a user.
func LoadCourseRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]CourseRow, error) {
	rows, err := pool.Query(ctx, `
SELECT
    c.id,
    ce.id,
    c.course_code,
    c.title,
    COALESCE(
        (SELECT cs.credits
         FROM catalog.catalog_sections cs
         WHERE cs.lms_course_id = c.id
         ORDER BY cs.updated_at DESC NULLS LAST
         LIMIT 1),
        0
    )::float8 AS credits,
    (
        SELECT fgs.final_grade
        FROM course.final_grade_submissions fgs
        WHERE fgs.enrollment_id = ce.id
        ORDER BY fgs.submitted_at DESC
        LIMIT 1
    ) AS final_grade,
    c.term_id,
    tr.name,
    tr.start_date,
    tr.end_date,
    COALESCE(ce.state::text, 'active') AS enrollment_state
FROM course.course_enrollments ce
JOIN course.courses c ON c.id = ce.course_id
LEFT JOIN tenant.terms tr ON tr.id = c.term_id
LEFT JOIN course.enrollment_roles er ON er.role_key = ce.role
WHERE ce.user_id = $1
  AND ce.active = TRUE
  AND (er.is_student_equivalent IS TRUE OR ce.role = 'student')
ORDER BY tr.start_date NULLS LAST, c.course_code
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseRow
	for rows.Next() {
		var r CourseRow
		if err := rows.Scan(
			&r.CourseID, &r.EnrollmentID, &r.CourseCode, &r.Title, &r.Credits,
			&r.FinalGrade, &r.TermID, &r.TermName, &r.TermStart, &r.TermEnd, &r.EnrollmentState,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Assemble builds a canonical academic record from loaded course rows.
func Assemble(params AssembleParams, rows []CourseRow) (*AcademicRecord, error) {
	if params.Variant == "" {
		params.Variant = VariantUnofficial
	}
	if params.Scale == "" {
		params.Scale = ScaleFourPoint
	}
	genAt := params.GeneratedAt.UTC().Truncate(time.Second)
	if genAt.IsZero() {
		genAt = time.Now().UTC().Truncate(time.Second)
	}

	termFilter := map[uuid.UUID]struct{}{}
	for _, id := range params.TermIDs {
		termFilter[id] = struct{}{}
	}

	type termAcc struct {
		block TermBlock
		sort  time.Time
	}
	byTerm := map[string]*termAcc{}
	hasIP := false

	for _, row := range rows {
		if len(termFilter) > 0 {
			if row.TermID == nil {
				continue
			}
			if _, ok := termFilter[*row.TermID]; !ok {
				continue
			}
		}

		grade, inProgress := resolveGrade(row, params.Variant)
		if inProgress {
			hasIP = true
		}
		// Official/final omit in-progress lines unless variant is in_progress or unofficial/partial.
		if inProgress && params.Variant == VariantOfficial {
			continue
		}

		credits := row.Credits
		if credits < 0 {
			credits = 0
		}
		line := CourseLine{
			Code:             strings.TrimSpace(row.CourseCode),
			Title:            strings.TrimSpace(row.Title),
			CreditsAttempted: round2(credits),
			InProgress:       inProgress,
		}
		if inProgress {
			line.Grade = "IP"
			line.CreditsEarned = 0
		} else {
			line.Grade = grade
			line.CreditsEarned = CreditsEarnedForGrade(grade, credits)
		}

		termKey := "none"
		label := "No term"
		var started, ended string
		var sortAt time.Time
		if row.TermID != nil {
			termKey = row.TermID.String()
		}
		if row.TermName != nil && strings.TrimSpace(*row.TermName) != "" {
			label = strings.TrimSpace(*row.TermName)
		}
		if row.TermStart != nil {
			started = row.TermStart.UTC().Format("2006-01-02")
			sortAt = row.TermStart.UTC()
		}
		if row.TermEnd != nil {
			ended = row.TermEnd.UTC().Format("2006-01-02")
		}

		acc, ok := byTerm[termKey]
		if !ok {
			tb := TermBlock{
				Label:     label,
				StartedOn: started,
				EndedOn:   ended,
				Courses:   nil,
			}
			if row.TermID != nil {
				tb.TermID = row.TermID.String()
			}
			acc = &termAcc{block: tb, sort: sortAt}
			byTerm[termKey] = acc
		}
		acc.block.Courses = append(acc.block.Courses, line)
	}

	terms := make([]TermBlock, 0, len(byTerm))
	order := make([]string, 0, len(byTerm))
	for k := range byTerm {
		order = append(order, k)
	}
	sort.Slice(order, func(i, j int) bool {
		a, b := byTerm[order[i]], byTerm[order[j]]
		if !a.sort.Equal(b.sort) {
			return a.sort.Before(b.sort)
		}
		return a.block.Label < b.block.Label
	})
	for _, k := range order {
		acc := byTerm[k]
		// Sort courses within term for byte-stable output.
		sort.Slice(acc.block.Courses, func(i, j int) bool {
			if acc.block.Courses[i].Code != acc.block.Courses[j].Code {
				return acc.block.Courses[i].Code < acc.block.Courses[j].Code
			}
			return acc.block.Courses[i].Title < acc.block.Courses[j].Title
		})
		_ = ComputeCumulative(acc.block.Courses, params.Scale)
		gpa, credits := ComputeTermGPA(acc.block.Courses, params.Scale)
		acc.block.TermGPA = gpa
		acc.block.TermCredits = credits
		terms = append(terms, acc.block)
	}

	allLines := make([]CourseLine, 0)
	for ti := range terms {
		allLines = append(allLines, terms[ti].Courses...)
	}
	cum := ComputeCumulative(allLines, params.Scale)
	// Propagate quality points computed on the flat list back onto term lines.
	idx := 0
	for ti := range terms {
		for ci := range terms[ti].Courses {
			if idx < len(allLines) {
				terms[ti].Courses[ci] = allLines[idx]
			}
			idx++
		}
	}

	inst := strings.TrimSpace(params.InstitutionName)
	if inst == "" {
		inst = "Lextures"
	}
	name := strings.TrimSpace(params.StudentName)
	if name == "" {
		name = "Student"
	}

	rec := &AcademicRecord{
		SchemaVersion:   SchemaVersion,
		TemplateVersion: TemplateVersion,
		Variant:         params.Variant,
		GeneratedAt:     genAt.Format(time.RFC3339),
		Student: StudentBlock{
			Name:      name,
			StudentID: strings.TrimSpace(params.StudentID),
		},
		Institution: InstitutionBlock{Name: inst},
		Terms:       terms,
		Cumulative:  cum,
		Legend:      DefaultLegend(),
		HasInProgress: hasIP && params.Variant != VariantOfficial,
	}
	if params.Variant == VariantPartial && len(params.TermIDs) == 0 {
		return nil, fmt.Errorf("academicrecord: partial variant requires termIds")
	}
	return rec, nil
}

// AssembleFromDB loads enrollments and builds the canonical record.
func AssembleFromDB(ctx context.Context, pool *pgxpool.Pool, params AssembleParams) (*AcademicRecord, error) {
	rows, err := LoadCourseRows(ctx, pool, params.UserID)
	if err != nil {
		return nil, err
	}
	return Assemble(params, rows)
}

func resolveGrade(row CourseRow, variant Variant) (grade string, inProgress bool) {
	state := strings.ToLower(strings.TrimSpace(row.EnrollmentState))
	switch state {
	case "withdrawn", "dropped":
		return "W", false
	case "audit":
		return "AU", false
	case "no_credit":
		return "NC", false
	case "incomplete":
		return "I", false
	}
	if row.FinalGrade != nil && strings.TrimSpace(*row.FinalGrade) != "" {
		return strings.TrimSpace(*row.FinalGrade), false
	}
	if variant == VariantOfficial {
		// Official omits ungraded; treat as in-progress so Assemble skips.
		return "IP", true
	}
	return "IP", true
}
