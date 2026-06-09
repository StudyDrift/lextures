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

// LibraryResourceMeta is the metadata stored with a library resource module item.
type LibraryResourceMeta struct {
	Title      string  `json:"title,omitempty"`
	Author     string  `json:"author,omitempty"`
	ISSN       string  `json:"issn,omitempty"`
	ISBN       string  `json:"isbn,omitempty"`
	Source     string  `json:"source,omitempty"`
	AlmaMmsID  string  `json:"almaMmsId,omitempty"`
	LegantoID  string  `json:"legantoListId,omitempty"`
	EZProxyURL string  `json:"ezproxyUrl,omitempty"`
}

// InsertLibraryResourceUnderModule appends a library_resource structure item and a
// module_library_resources row in a single transaction.
func InsertLibraryResourceUnderModule(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, moduleID uuid.UUID,
	title string,
	resourceType string,
	externalToolID *uuid.UUID,
	meta LibraryResourceMeta,
	ezproxyURL string,
) (ItemRow, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return ItemRow{}, errors.New("coursestructure: library resource title is required")
	}
	rt := strings.TrimSpace(resourceType)
	if rt != "catalog_item" && rt != "leganto_list" {
		rt = "catalog_item"
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return ItemRow{}, err
	}
	ezp := strings.TrimSpace(ezproxyURL)
	var ezpPtr *string
	if ezp != "" {
		ezpPtr = &ezp
	}

	return insertModuleChild(ctx, pool, courseID, moduleID, "library_resource", t, func(tx pgx.Tx, itemID uuid.UUID) error {
		_, err := tx.Exec(ctx, `
INSERT INTO course.module_library_resources
    (structure_item_id, resource_type, external_tool_id, alma_mms_id, leganto_list_id, metadata, ezproxy_url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`,
			itemID,
			rt,
			externalToolID,
			nilIfEmptyStr(meta.AlmaMmsID),
			nilIfEmptyStr(meta.LegantoID),
			metaBytes,
			ezpPtr,
		)
		return err
	})
}

func nilIfEmptyStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := strings.TrimSpace(s)
	return &v
}
