// Package quizgame persists course-scoped live quiz kits (plan IQ.1).
package quizgame

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

const maxTitleLen = 200
const maxSlugAttempts = 8
const defaultPageSize = 50
const maxPageSize = 100

// Kit is one quiz kit row.
type Kit struct {
	ID            string    `json:"id"`
	CourseID      string    `json:"courseId"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Slug          string    `json:"slug"`
	CoverImageRef *string   `json:"coverImageRef"`
	Status        string    `json:"status"`
	Visibility    string    `json:"visibility"`
	Tags          []string  `json:"tags"`
	QuestionCount int       `json:"questionCount"`
	Archived      bool      `json:"archived"`
	CreatedBy     *string   `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ListOpts filters and paginates kit listings.
type ListOpts struct {
	Query           string
	Tag             string
	IncludeArchived bool
	Page            int // 1-based
	PageSize        int
}

// ListResult is a page of kits plus total matching count.
type ListResult struct {
	Kits       []Kit
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// PatchKitInput is a partial update for kit metadata.
type PatchKitInput struct {
	Title         *string
	Description   *string
	CoverImageRef *string
	Status        *string
	Visibility    *string
	Tags          *[]string
	Archived      *bool
}

func scanKit(row pgx.Row) (Kit, error) {
	var k Kit
	var id, courseID uuid.UUID
	var createdBy uuid.NullUUID
	var cover *string
	var tags []string
	if err := row.Scan(
		&id, &courseID, &k.Title, &k.Description, &k.Slug, &cover,
		&k.Status, &k.Visibility, &tags, &k.QuestionCount, &k.Archived,
		&createdBy, &k.CreatedAt, &k.UpdatedAt,
	); err != nil {
		return Kit{}, err
	}
	k.ID = id.String()
	k.CourseID = courseID.String()
	k.CoverImageRef = cover
	if tags == nil {
		tags = []string{}
	}
	k.Tags = tags
	if createdBy.Valid {
		s := createdBy.UUID.String()
		k.CreatedBy = &s
	}
	return k, nil
}

func selectKitCols() string {
	return `k.id, k.course_id, k.title, k.description, k.slug, k.cover_image_ref,
		k.status::text, k.visibility::text, k.tags, k.question_count, k.archived,
		k.created_by, k.created_at, k.updated_at`
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// List returns kits for a course, newest updated first, with optional filters.
func List(ctx context.Context, pool *pgxpool.Pool, courseCode string, opts ListOpts) (*ListResult, error) {
	page, pageSize := normalizePage(opts.Page, opts.PageSize)
	q := strings.TrimSpace(opts.Query)
	tag := strings.TrimSpace(opts.Tag)

	where := `WHERE c.course_code = $1`
	args := []any{courseCode}
	argN := 2
	if !opts.IncludeArchived {
		where += ` AND k.archived = FALSE`
	}
	if q != "" {
		where += fmt.Sprintf(` AND k.title ILIKE $%d`, argN)
		args = append(args, "%"+q+"%")
		argN++
	}
	if tag != "" {
		where += fmt.Sprintf(` AND $%d = ANY(k.tags)`, argN)
		args = append(args, tag)
		argN++
	}

	var total int
	countQ := `
		SELECT COUNT(*)
		FROM quizgame.kits k
		INNER JOIN course.courses c ON c.id = k.course_id
		` + where
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	listQ := `
		SELECT ` + selectKitCols() + `
		FROM quizgame.kits k
		INNER JOIN course.courses c ON c.id = k.course_id
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
	return &ListResult{
		Kits:       out,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Get returns a single kit by id within a course, or nil if not found.
func Get(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) (*Kit, error) {
	id, err := uuid.Parse(kitID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectKitCols()+`
		FROM quizgame.kits k
		INNER JOIN course.courses c ON c.id = k.course_id
		WHERE c.course_code = $1 AND k.id = $2
	`, courseCode, id)
	k, err := scanKit(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

// Create inserts a kit with a unique slug derived from title.
func Create(ctx context.Context, pool *pgxpool.Pool, courseCode string, createdBy uuid.UUID, title, description string, tags []string) (*Kit, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("quizgame: title is required")
	}
	if len(title) > maxTitleLen {
		return nil, fmt.Errorf("quizgame: title must be at most %d characters", maxTitleLen)
	}
	if tags == nil {
		tags = []string{}
	}

	baseSlug := organization.SuggestSlugFromName(title)
	if baseSlug == "" {
		baseSlug = "kit"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var courseID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM course.courses WHERE course_code = $1
	`, courseCode).Scan(&courseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var created *Kit
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := baseSlug
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
			if len(slug) > 48 {
				slug = slug[:48]
			}
		}
		// Unique-violation retries must use a savepoint; otherwise the TX is aborted.
		sp := fmt.Sprintf("kit_slug_%d", attempt)
		if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
			return nil, err
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO quizgame.kits (course_id, title, description, slug, tags, created_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, course_id, title, description, slug, cover_image_ref,
				status::text, visibility::text, tags, question_count, archived,
				created_by, created_at, updated_at
		`, courseID, title, description, slug, tags, createdBy)
		k, err := scanKit(row)
		if err == nil {
			if _, relErr := tx.Exec(ctx, "RELEASE SAVEPOINT "+sp); relErr != nil {
				return nil, relErr
			}
			created = &k
			break
		}
		if _, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp); rbErr != nil {
			return nil, rbErr
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
	}
	if created == nil {
		return nil, fmt.Errorf("quizgame: could not allocate unique slug")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

// Patch updates kit metadata fields.
func Patch(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string, in PatchKitInput) (*Kit, error) {
	id, err := uuid.Parse(kitID)
	if err != nil {
		return nil, nil
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return nil, fmt.Errorf("quizgame: title is required")
		}
		if len(t) > maxTitleLen {
			return nil, fmt.Errorf("quizgame: title must be at most %d characters", maxTitleLen)
		}
		in.Title = &t
	}
	if in.Status != nil {
		s := strings.TrimSpace(strings.ToLower(*in.Status))
		switch s {
		case "draft", "ready", "archived":
			in.Status = &s
		default:
			return nil, fmt.Errorf("quizgame: invalid status")
		}
	}
	if in.Visibility != nil {
		v := strings.TrimSpace(strings.ToLower(*in.Visibility))
		switch v {
		case "private", "course", "org", "public":
			in.Visibility = &v
		default:
			return nil, fmt.Errorf("quizgame: invalid visibility")
		}
	}

	var tags any
	if in.Tags != nil {
		t := *in.Tags
		if t == nil {
			t = []string{}
		}
		tags = t
	}

	row := pool.QueryRow(ctx, `
		UPDATE quizgame.kits k
		SET
			title = COALESCE($3, k.title),
			description = COALESCE($4, k.description),
			cover_image_ref = COALESCE($5, k.cover_image_ref),
			status = COALESCE($6::quizgame.kit_status, k.status),
			visibility = COALESCE($7::quizgame.kit_visibility, k.visibility),
			tags = COALESCE($8, k.tags),
			archived = COALESCE($9, k.archived),
			updated_at = NOW()
		FROM course.courses c
		WHERE c.id = k.course_id AND c.course_code = $1 AND k.id = $2
		RETURNING k.id, k.course_id, k.title, k.description, k.slug, k.cover_image_ref,
			k.status::text, k.visibility::text, k.tags, k.question_count, k.archived,
			k.created_by, k.created_at, k.updated_at
	`, courseCode, id, in.Title, in.Description, in.CoverImageRef, in.Status, in.Visibility, tags, in.Archived)
	k, err := scanKit(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

// Archive soft-archives a kit (sets archived=true and status=archived).
func Archive(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) (*Kit, error) {
	archived := true
	status := "archived"
	return Patch(ctx, pool, courseCode, kitID, PatchKitInput{Archived: &archived, Status: &status})
}

// Restore un-archives a kit (sets archived=false and status=draft).
func Restore(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) (*Kit, error) {
	archived := false
	status := "draft"
	return Patch(ctx, pool, courseCode, kitID, PatchKitInput{Archived: &archived, Status: &status})
}

// Duplicate creates a metadata-only copy of a kit (no questions in IQ.1).
func Duplicate(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string, createdBy uuid.UUID) (*Kit, error) {
	src, err := Get(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}
	title := src.Title + " (copy)"
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen]
	}
	return Create(ctx, pool, courseCode, createdBy, title, src.Description, src.Tags)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "idx_quizgame_active_join_code") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "23505")
}
