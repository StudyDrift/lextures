package coursestructure

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TextbookResourceMeta is the metadata stored with a textbook resource module item.
// It mirrors the VitalSource / RedShelf deep-link descriptor.
type TextbookResourceMeta struct {
	ISBN      string `json:"isbn,omitempty"`
	Title     string `json:"title,omitempty"`
	Edition   string `json:"edition,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	Chapter   string `json:"chapter,omitempty"`
	PageRange string `json:"pageRange,omitempty"`
}

// InsertTextbookResourceUnderModule appends a textbook_resource structure item and a
// module_textbook_resources row in a single transaction.
func InsertTextbookResourceUnderModule(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, moduleID uuid.UUID,
	title string,
	provider string,
	externalToolID *uuid.UUID,
	meta TextbookResourceMeta,
) (ItemRow, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return ItemRow{}, errors.New("coursestructure: textbook resource title is required")
	}
	p := strings.ToLower(strings.TrimSpace(provider))
	if p != "vitalsource" && p != "redshelf" {
		p = "vitalsource"
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return ItemRow{}, err
	}

	return insertModuleChild(ctx, pool, courseID, moduleID, "textbook_resource", t, func(tx pgx.Tx, itemID uuid.UUID) error {
		_, err := tx.Exec(ctx, `
INSERT INTO course.module_textbook_resources
    (structure_item_id, provider, external_tool_id, metadata)
VALUES ($1, $2, $3, $4)
`,
			itemID,
			p,
			externalToolID,
			metaBytes,
		)
		return err
	})
}
