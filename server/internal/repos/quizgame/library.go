package quizgame

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

// LibraryOpts filters the discovery library.
type LibraryOpts struct {
	Query              string
	Subject            string
	GradeBand          string
	Language           string
	Tag                string
	IncludePublicCatalog bool
	Viewer             uuid.UUID
	Page               int
	PageSize           int
}

// LibraryResult is a page of discoverable kits.
type LibraryResult struct {
	Kits       []Kit
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// SearchLibrary returns kits shared with the viewer (and optionally listed public catalog).
func SearchLibrary(ctx context.Context, pool *pgxpool.Pool, opts LibraryOpts) (*LibraryResult, error) {
	page, pageSize := normalizePage(opts.Page, opts.PageSize)
	orgID, _ := organization.OrgIDForUser(ctx, pool, opts.Viewer)

	where := `
		WHERE k.archived = FALSE
		  AND k.is_template = FALSE
		  AND (
			-- Shared with viewer
			EXISTS (
				SELECT 1 FROM quizgame.kit_shares s
				WHERE s.kit_id = k.id
				  AND (
					(s.grantee_type = 'user' AND s.grantee_id = $1)
					OR (s.grantee_type = 'course' AND s.grantee_id IN (
						SELECT course_id FROM course.course_enrollments WHERE user_id = $1 AND active
					))
					OR (s.grantee_type = 'org_unit' AND s.grantee_id IN (
						SELECT c2.org_unit_id FROM course.course_enrollments e
						INNER JOIN course.courses c2 ON c2.id = e.course_id
						WHERE e.user_id = $1 AND e.active AND c2.org_unit_id IS NOT NULL
					))
					OR (s.grantee_type = 'org' AND s.grantee_id IS NULL AND $2::uuid IS NOT NULL
						AND c.org_id = $2)
				  )
				  AND s.permission IN ('view', 'copy', 'edit')
			)
			-- Org-visible kits in same org
			OR (k.visibility = 'org' AND $2::uuid IS NOT NULL AND c.org_id = $2)
	`
	args := []any{opts.Viewer, nullUUID(orgID)}
	argN := 3

	if opts.IncludePublicCatalog {
		where += ` OR (k.catalog_status = 'listed' AND k.visibility = 'public')`
	}
	where += `)`

	q := strings.TrimSpace(opts.Query)
	if q != "" {
		where += fmt.Sprintf(` AND (
			to_tsvector('english', coalesce(k.title,'') || ' ' || coalesce(k.description,''))
				@@ plainto_tsquery('english', $%d)
			OR k.title ILIKE $%d
		)`, argN, argN+1)
		args = append(args, q, "%"+q+"%")
		argN += 2
	}
	if s := strings.TrimSpace(opts.Subject); s != "" {
		where += fmt.Sprintf(` AND k.subject ILIKE $%d`, argN)
		args = append(args, s)
		argN++
	}
	if g := strings.TrimSpace(opts.GradeBand); g != "" {
		where += fmt.Sprintf(` AND k.grade_band ILIKE $%d`, argN)
		args = append(args, g)
		argN++
	}
	if lang := strings.TrimSpace(opts.Language); lang != "" {
		where += fmt.Sprintf(` AND k.language ILIKE $%d`, argN)
		args = append(args, lang)
		argN++
	}
	if tag := strings.TrimSpace(opts.Tag); tag != "" {
		where += fmt.Sprintf(` AND $%d = ANY(k.tags)`, argN)
		args = append(args, tag)
		argN++
	}

	var total int
	countQ := `
		SELECT COUNT(*)
		FROM quizgame.kits k
		LEFT JOIN course.courses c ON c.id = k.course_id
		` + where
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	listQ := `
		SELECT ` + selectKitCols() + `
		FROM quizgame.kits k
		LEFT JOIN course.courses c ON c.id = k.course_id
		` + where + `
		ORDER BY k.updated_at DESC
		LIMIT $` + fmt.Sprintf("%d", argN) + ` OFFSET $` + fmt.Sprintf("%d", argN+1)
	args = append(args, pageSize, offset)

	rows, err := pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Kit, 0)
	for rows.Next() {
		k, err := scanKit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return &LibraryResult{
		Kits:       out,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// ImportFromLibrary deep-copies a library kit into a target course and re-validates.
func ImportFromLibrary(
	ctx context.Context,
	pool *pgxpool.Pool,
	kitID, targetCourseCode string,
	createdBy uuid.UUID,
) (*Kit, *ValidateResult, error) {
	src, err := GetByID(ctx, pool, kitID)
	if err != nil {
		return nil, nil, err
	}
	if src == nil {
		return nil, nil, nil
	}
	ok, err := CanAccessKit(ctx, pool, src, createdBy, SharePermCopy)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, fmt.Errorf("quizgame: not allowed to import this kit")
	}

	attr := src.Attribution
	if attr == "" {
		attr = "Imported from \"" + src.Title + "\""
	}
	copied, err := DeepCopyKit(ctx, pool, DeepCopyOpts{
		SourceKitID:      kitID,
		TargetCourseCode: targetCourseCode,
		CreatedBy:        createdBy,
		Title:            src.Title,
		DropBankLinks:    true,
		CopyCatalogMeta:  true,
		Attribution:      attr,
	})
	if err != nil || copied == nil {
		return copied, nil, err
	}

	vr, err := ValidateKit(ctx, pool, targetCourseCode, copied.ID)
	if err != nil {
		return copied, nil, err
	}
	return copied, vr, nil
}

// SubmitToCatalog sets catalog_status to pending (awaits moderation / IQ.9).
func SubmitToCatalog(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) (*Kit, error) {
	id, err := uuid.Parse(kitID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE quizgame.kits k
		SET catalog_status = 'pending',
			visibility = 'public',
			updated_at = NOW()
		FROM course.courses c
		WHERE c.id = k.course_id AND c.course_code = $1 AND k.id = $2
		  AND k.is_template = FALSE
		RETURNING `+selectKitCols(),
		courseCode, id)
	k, err := scanKit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// SetCatalogStatus is used by moderation (admin) to approve/reject.
func SetCatalogStatus(ctx context.Context, pool *pgxpool.Pool, kitID, status string) (*Kit, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "unlisted", "pending", "listed", "rejected":
	default:
		return nil, fmt.Errorf("quizgame: invalid catalog status")
	}
	id, err := uuid.Parse(kitID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE quizgame.kits k
		SET catalog_status = $2, updated_at = NOW()
		WHERE k.id = $1
		RETURNING `+selectKitCols(),
		id, status)
	k, err := scanKit(row)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// PreviewKit returns a kit when the viewer has at least view access.
func PreviewKit(ctx context.Context, pool *pgxpool.Pool, kitID string, viewer uuid.UUID) (*Kit, []Question, error) {
	kit, err := GetByID(ctx, pool, kitID)
	if err != nil || kit == nil {
		return nil, nil, err
	}
	ok, err := CanAccessKit(ctx, pool, kit, viewer, SharePermView)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, fmt.Errorf("quizgame: not allowed to preview this kit")
	}

	courseCode, err := CourseCodeForKitID(ctx, pool, kitID)
	if err != nil {
		return nil, nil, err
	}
	var questions []Question
	if courseCode != "" {
		questions, err = ListQuestions(ctx, pool, courseCode, kitID)
	} else {
		// System / course-less kits: load by kit id directly.
		questions, err = listQuestionsByKitID(ctx, pool, kitID)
	}
	if err != nil {
		return nil, nil, err
	}
	return kit, questions, nil
}

func listQuestionsByKitID(ctx context.Context, pool *pgxpool.Pool, kitID string) ([]Question, error) {
	kid, err := uuid.Parse(kitID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT `+selectQuestionCols()+`
		FROM quizgame.questions q
		WHERE q.kit_id = $1
		ORDER BY q.position ASC
	`, kid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Question, 0)
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}
