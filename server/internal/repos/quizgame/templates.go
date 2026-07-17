package quizgame

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

const (
	TemplateScopeSystem = "system"
	TemplateScopeOrg    = "org"
	TemplateScopeCourse = "course"
)

// ListTemplatesFilter scopes the New-from-template picker.
type ListTemplatesFilter struct {
	Scope      string // system|org|course|"" (all visible)
	CourseCode string
	OrgID      *uuid.UUID
	Query      string
}

// ListTemplates returns template kits visible to the caller.
func ListTemplates(ctx context.Context, pool *pgxpool.Pool, f ListTemplatesFilter) ([]Kit, error) {
	q := `
		SELECT ` + selectKitCols() + `
		FROM quizgame.kits k
		LEFT JOIN course.courses c ON c.id = k.course_id
		WHERE k.is_template = TRUE AND k.archived = FALSE`
	args := make([]any, 0, 6)
	n := 1

	scope := strings.TrimSpace(strings.ToLower(f.Scope))
	switch scope {
	case TemplateScopeSystem:
		q += ` AND k.template_scope = 'system'`
	case TemplateScopeCourse:
		if strings.TrimSpace(f.CourseCode) == "" {
			return nil, fmt.Errorf("quizgame: courseCode is required for course scope")
		}
		q += fmt.Sprintf(` AND k.template_scope = 'course' AND c.course_code = $%d`, n)
		args = append(args, f.CourseCode)
		n++
	case TemplateScopeOrg:
		if f.OrgID == nil {
			return []Kit{}, nil
		}
		q += fmt.Sprintf(` AND k.template_scope = 'org' AND c.org_id = $%d`, n)
		args = append(args, *f.OrgID)
		n++
	case "":
		q += ` AND (`
		q += ` k.template_scope = 'system'`
		if strings.TrimSpace(f.CourseCode) != "" {
			q += fmt.Sprintf(` OR (k.template_scope = 'course' AND c.course_code = $%d)`, n)
			args = append(args, f.CourseCode)
			n++
		}
		if f.OrgID != nil {
			q += fmt.Sprintf(` OR (k.template_scope = 'org' AND c.org_id = $%d)`, n)
			args = append(args, *f.OrgID)
			n++
		}
		q += `)`
	default:
		return nil, fmt.Errorf("quizgame: invalid template scope")
	}

	search := strings.TrimSpace(f.Query)
	if search != "" {
		q += fmt.Sprintf(` AND (k.title ILIKE $%d OR k.description ILIKE $%d)`, n, n)
		args = append(args, "%"+search+"%")
		n++
	}

	q += ` ORDER BY
		CASE k.template_scope WHEN 'system' THEN 0 WHEN 'org' THEN 1 ELSE 2 END,
		k.title ASC`

	rows, err := pool.Query(ctx, q, args...)
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
	return out, rows.Err()
}

// SaveAsTemplateInput marks a deep-copied kit as a reusable template.
type SaveAsTemplateInput struct {
	Scope       string
	Title       string
	Description string
	Tags        []string
	OrgID       *uuid.UUID
}

// SaveAsTemplate deep-copies a kit into a template (course or org scope).
func SaveAsTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, kitID string,
	createdBy uuid.UUID,
	in SaveAsTemplateInput,
) (*Kit, error) {
	scope := strings.TrimSpace(strings.ToLower(in.Scope))
	switch scope {
	case TemplateScopeCourse, TemplateScopeOrg:
	default:
		return nil, fmt.Errorf("quizgame: scope must be course or org")
	}
	if scope == TemplateScopeOrg && (in.OrgID == nil || *in.OrgID == uuid.Nil) {
		return nil, fmt.Errorf("quizgame: org scope requires organization")
	}

	src, err := Get(ctx, pool, courseCode, kitID)
	if err != nil || src == nil {
		return nil, err
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = src.Title
	}
	desc := src.Description
	if strings.TrimSpace(in.Description) != "" {
		desc = in.Description
	}
	tags := src.Tags
	if in.Tags != nil {
		tags = in.Tags
	}

	copied, err := DeepCopyKit(ctx, pool, DeepCopyOpts{
		SourceKitID:      kitID,
		TargetCourseCode: courseCode,
		CreatedBy:        createdBy,
		Title:            title,
		AsTemplate:       true,
		TemplateScope:    scope,
		CopyCatalogMeta:  true,
	})
	if err != nil || copied == nil {
		return copied, err
	}

	// Apply description/tags overrides after copy.
	id, err := uuid.Parse(copied.ID)
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []string{}
	}
	row := pool.QueryRow(ctx, `
		UPDATE quizgame.kits k
		SET description = $2, tags = $3, updated_at = NOW()
		WHERE k.id = $1
		RETURNING `+selectKitCols(),
		id, desc, tags)
	k, err := scanKit(row)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// CreateKitFromTemplate duplicates a template into a target course as an editable kit.
func CreateKitFromTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	templateID, targetCourseCode string,
	createdBy uuid.UUID,
) (*Kit, error) {
	tmpl, err := GetByID(ctx, pool, templateID)
	if err != nil {
		return nil, err
	}
	if tmpl == nil || !tmpl.IsTemplate {
		return nil, nil
	}
	title := tmpl.Title
	return DeepCopyKit(ctx, pool, DeepCopyOpts{
		SourceKitID:      templateID,
		TargetCourseCode: targetCourseCode,
		CreatedBy:        createdBy,
		Title:            title,
		DropBankLinks:    true,
		CopyCatalogMeta:  true,
		Attribution:      "From template \"" + tmpl.Title + "\"",
	})
}

// NormalizeTemplateScope validates template scope for API handlers.
func NormalizeTemplateScope(scope string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(scope))
	switch s {
	case TemplateScopeSystem, TemplateScopeOrg, TemplateScopeCourse:
		return s, nil
	default:
		return "", fmt.Errorf("quizgame: invalid template scope")
	}
}

// ResolveOrgIDForTemplates returns the caller's org id when present.
func ResolveOrgIDForTemplates(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*uuid.UUID, error) {
	id, err := organization.OrgIDForUser(ctx, pool, userID)
	if err != nil || id == uuid.Nil {
		return nil, err
	}
	return &id, nil
}
