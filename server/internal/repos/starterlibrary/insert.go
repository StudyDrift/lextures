// Package starterlibrary writes a published starter outline into an existing course.
package starterlibrary

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// Page is one content page to insert under a module.
type Page struct {
	Title    string
	Markdown string
}

// Module is a module and its pages.
type Module struct {
	Title string
	Pages []Page
}

// InsertPublishedOutline inserts modules and content pages as published in one transaction.
func InsertPublishedOutline(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, modules []Module) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if courseID == uuid.Nil {
		return errors.New("course id is required")
	}
	if len(modules) == 0 {
		return errors.New("at least one module is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for mi, mod := range modules {
		title := strings.TrimSpace(mod.Title)
		if title == "" {
			return fmt.Errorf("module %d: title is required", mi+1)
		}
		if err := coursestructure.ValidateItemTitle(title); err != nil {
			return err
		}
		if len(mod.Pages) == 0 {
			return fmt.Errorf("module %q: at least one page is required", title)
		}

		var moduleID uuid.UUID
		if err := tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (
	course_id, sort_order, kind, title, parent_id, published, visible_from, archived
) VALUES ($1, $2, 'module', $3, NULL, true, NULL, false)
RETURNING id
`, courseID, mi, title).Scan(&moduleID); err != nil {
			return fmt.Errorf("insert module %q: %w", title, err)
		}

		for pi, page := range mod.Pages {
			pageTitle := strings.TrimSpace(page.Title)
			if pageTitle == "" {
				return fmt.Errorf("module %q page %d: title is required", title, pi+1)
			}
			if err := coursestructure.ValidateItemTitle(pageTitle); err != nil {
				return err
			}
			var pageID uuid.UUID
			if err := tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (
	course_id, sort_order, kind, title, parent_id, published, visible_from, archived
) VALUES ($1, $2, 'content_page', $3, $4, true, NULL, false)
RETURNING id
`, courseID, pi, pageTitle, moduleID).Scan(&pageID); err != nil {
				return fmt.Errorf("insert page %q: %w", pageTitle, err)
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO course.module_content_pages (structure_item_id, markdown)
VALUES ($1, $2)
`, pageID, page.Markdown); err != nil {
				return fmt.Errorf("insert page body %q: %w", pageTitle, err)
			}
		}
	}

	return tx.Commit(ctx)
}
