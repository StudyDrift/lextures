package board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Template is a row in board.board_templates.
type Template struct {
	ID          string          `json:"id"`
	Scope       string          `json:"scope"`
	CourseID    *string         `json:"courseId,omitempty"`
	OrgID       *string         `json:"orgId,omitempty"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Tags        []string        `json:"tags"`
	Definition  json.RawMessage `json:"definition"`
	CreatedBy   *string         `json:"createdBy,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
}

// ListTemplatesFilter scopes gallery listing (FR-6).
type ListTemplatesFilter struct {
	Scope      string // builtin|course|org|"" (all visible)
	CourseCode string
	OrgID      *uuid.UUID
	Query      string
	Locale     string
}

func scanTemplate(row pgx.Row) (Template, error) {
	var t Template
	var id uuid.UUID
	var courseID, orgID, createdBy uuid.NullUUID
	var tags []string
	var def []byte
	if err := row.Scan(
		&id, &t.Scope, &courseID, &orgID, &t.Title, &t.Description, &tags, &def, &createdBy, &t.CreatedAt,
	); err != nil {
		return Template{}, err
	}
	t.ID = id.String()
	if courseID.Valid {
		s := courseID.UUID.String()
		t.CourseID = &s
	}
	if orgID.Valid {
		s := orgID.UUID.String()
		t.OrgID = &s
	}
	if createdBy.Valid {
		s := createdBy.UUID.String()
		t.CreatedBy = &s
	}
	if tags == nil {
		tags = []string{}
	}
	t.Tags = tags
	if len(def) > 0 {
		t.Definition = json.RawMessage(def)
	} else {
		t.Definition = json.RawMessage(`{}`)
	}
	return t, nil
}

func selectTemplateCols() string {
	return `t.id, t.scope, t.course_id, t.org_id, t.title, t.description, t.tags, t.definition, t.created_by, t.created_at`
}

// ListTemplates returns gallery templates filtered by scope and search (FR-6).
func ListTemplates(ctx context.Context, pool *pgxpool.Pool, f ListTemplatesFilter) ([]Template, error) {
	q := `
		SELECT ` + selectTemplateCols() + `
		FROM board.board_templates t
		LEFT JOIN course.courses c ON c.id = t.course_id
		WHERE 1=1`
	args := make([]any, 0, 6)
	n := 1

	scope := strings.TrimSpace(strings.ToLower(f.Scope))
	switch scope {
	case TemplateScopeBuiltin:
		q += ` AND t.scope = 'builtin'`
	case TemplateScopeCourse:
		if strings.TrimSpace(f.CourseCode) == "" {
			return nil, fmt.Errorf("board: courseCode is required for course scope")
		}
		q += fmt.Sprintf(` AND t.scope = 'course' AND c.course_code = $%d`, n)
		args = append(args, f.CourseCode)
		n++
	case TemplateScopeOrg:
		if f.OrgID == nil {
			return []Template{}, nil
		}
		q += fmt.Sprintf(` AND t.scope = 'org' AND t.org_id = $%d`, n)
		args = append(args, *f.OrgID)
		n++
	case "":
		// All visible: builtins + this course + caller's org.
		q += ` AND (`
		q += ` t.scope = 'builtin'`
		if strings.TrimSpace(f.CourseCode) != "" {
			q += fmt.Sprintf(` OR (t.scope = 'course' AND c.course_code = $%d)`, n)
			args = append(args, f.CourseCode)
			n++
		}
		if f.OrgID != nil {
			q += fmt.Sprintf(` OR (t.scope = 'org' AND t.org_id = $%d)`, n)
			args = append(args, *f.OrgID)
			n++
		}
		q += `)`
	default:
		return nil, fmt.Errorf("board: invalid scope")
	}

	if qstr := strings.TrimSpace(f.Query); qstr != "" {
		q += fmt.Sprintf(` AND (
			t.title ILIKE $%d OR t.description ILIKE $%d OR EXISTS (
				SELECT 1 FROM unnest(t.tags) tag WHERE tag ILIKE $%d
			)
		)`, n, n, n)
		args = append(args, "%"+qstr+"%")
	}

	q += ` ORDER BY
		CASE t.scope WHEN 'builtin' THEN 0 WHEN 'course' THEN 1 ELSE 2 END,
		t.title ASC`

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		ApplyBuiltinLocale(&t, f.Locale)
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTemplate returns one template by id, or nil.
func GetTemplate(ctx context.Context, pool *pgxpool.Pool, templateID string) (*Template, error) {
	id, err := uuid.Parse(templateID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectTemplateCols()+`
		FROM board.board_templates t
		WHERE t.id = $1
	`, id)
	t, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// TemplateVisible reports whether the viewer may use this template in the given course/org.
func TemplateVisible(t Template, courseID, orgID *uuid.UUID) bool {
	switch t.Scope {
	case TemplateScopeBuiltin:
		return true
	case TemplateScopeCourse:
		if t.CourseID == nil || courseID == nil {
			return false
		}
		return *t.CourseID == courseID.String()
	case TemplateScopeOrg:
		if t.OrgID == nil || orgID == nil {
			return false
		}
		return *t.OrgID == orgID.String()
	default:
		return false
	}
}

// SaveAsTemplateInput creates a course or org template from a board (FR-5).
type SaveAsTemplateInput struct {
	Scope        string
	Title        string
	Description  string
	Tags         []string
	IncludePosts bool
	OrgID        *uuid.UUID // required for org scope
}

// SaveAsTemplate snapshots a board into board_templates.
func SaveAsTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	createdBy uuid.UUID,
	in SaveAsTemplateInput,
) (*Template, error) {
	scope, err := NormalizeTemplateScope(in.Scope)
	if err != nil {
		return nil, err
	}
	if scope == TemplateScopeBuiltin {
		return nil, fmt.Errorf("board: cannot save as builtin scope")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("board: title is required")
	}
	if len(title) > maxTitleLen {
		return nil, fmt.Errorf("board: title must be at most %d characters", maxTitleLen)
	}

	b, err := Get(ctx, pool, courseCode, boardID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	sections, err := ListSections(ctx, pool, courseCode, boardID)
	if err != nil {
		return nil, err
	}
	var posts []Post
	if in.IncludePosts {
		posts, err = ListPosts(ctx, pool, courseCode, boardID)
		if err != nil {
			return nil, err
		}
	}
	def := BoardToDefinition(*b, sections, posts, in.IncludePosts)
	raw, err := MarshalDefinition(def)
	if err != nil {
		return nil, err
	}

	courseUUID, err := uuid.Parse(b.CourseID)
	if err != nil {
		return nil, fmt.Errorf("board: invalid course id")
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	var insertedID uuid.UUID
	switch scope {
	case TemplateScopeCourse:
		err = pool.QueryRow(ctx, `
			INSERT INTO board.board_templates (
				scope, course_id, org_id, title, description, tags, definition, created_by
			) VALUES ('course', $1, NULL, $2, $3, $4, $5, $6)
			RETURNING id
		`, courseUUID, title, strings.TrimSpace(in.Description), tags, raw, createdBy).Scan(&insertedID)
	case TemplateScopeOrg:
		if in.OrgID == nil {
			return nil, fmt.Errorf("board: org_id is required for org scope")
		}
		err = pool.QueryRow(ctx, `
			INSERT INTO board.board_templates (
				scope, course_id, org_id, title, description, tags, definition, created_by
			) VALUES ('org', NULL, $1, $2, $3, $4, $5, $6)
			RETURNING id
		`, *in.OrgID, title, strings.TrimSpace(in.Description), tags, raw, createdBy).Scan(&insertedID)
	default:
		return nil, fmt.Errorf("board: invalid scope")
	}
	if err != nil {
		return nil, err
	}
	return GetTemplate(ctx, pool, insertedID.String())
}

// CourseOrgID returns the org_id for a course, if any.
func CourseOrgID(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*uuid.UUID, error) {
	var orgID uuid.NullUUID
	err := pool.QueryRow(ctx, `
		SELECT org_id FROM course.courses WHERE course_code = $1
	`, courseCode).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !orgID.Valid {
		return nil, nil
	}
	id := orgID.UUID
	return &id, nil
}

// CourseIDByCode returns the course UUID for a course code.
func CourseIDByCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT id FROM course.courses WHERE course_code = $1
	`, courseCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
