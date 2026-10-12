package starterlibrary

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/course"
	starterrepo "github.com/lextures/lextures/server/internal/repos/starterlibrary"
)

// CopyResult is the editable course created from a starter outline.
type CopyResult struct {
	CourseID   uuid.UUID
	CourseCode string
	Title      string
}

// CopyForUser creates a published, editable course owned by userID.
func CopyForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, outline Course) (*CopyResult, error) {
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user is required")
	}
	title := strings.TrimSpace(outline.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(outline.Modules) == 0 {
		return nil, fmt.Errorf("course has no modules")
	}

	description := strings.TrimSpace(outline.Summary)
	if len(outline.Standards) > 0 {
		description = strings.TrimSpace(description + "\n\nStandards: " + strings.Join(outline.Standards, ", ") + ".")
	}
	description = strings.TrimSpace(description + "\n\nCopied from the starter library. Edit any module to fit your family.")

	var grades []string
	if ValidBand(outline.GradeBand) {
		grades = []string{outline.GradeBand}
	}

	created, err := course.CreateCourse(ctx, pool, userID, title, description, "traditional", nil, nil, grades)
	if err != nil {
		return nil, err
	}
	courseID, err := uuid.Parse(created.ID)
	if err != nil {
		_ = course.DeleteByCode(ctx, pool, created.CourseCode)
		return nil, fmt.Errorf("created course id: %w", err)
	}

	modules := make([]starterrepo.Module, 0, len(outline.Modules))
	for _, mod := range outline.Modules {
		pages := make([]starterrepo.Page, 0, len(mod.Pages))
		for _, page := range mod.Pages {
			pages = append(pages, starterrepo.Page{Title: page.Title, Markdown: page.Markdown})
		}
		modules = append(modules, starterrepo.Module{Title: mod.Title, Pages: pages})
	}
	if err := starterrepo.InsertPublishedOutline(ctx, pool, courseID, modules); err != nil {
		_ = course.DeleteByCode(ctx, pool, created.CourseCode)
		return nil, err
	}
	if err := course.SetPublished(ctx, pool, created.CourseCode, true); err != nil {
		_ = course.DeleteByCode(ctx, pool, created.CourseCode)
		return nil, err
	}
	return &CopyResult{CourseID: courseID, CourseCode: created.CourseCode, Title: created.Title}, nil
}
