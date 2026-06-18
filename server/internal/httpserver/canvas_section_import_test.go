package httpserver

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCanvasSectionCodeFromRow_prefersSIS(t *testing.T) {
	row := map[string]any{
		"id":              float64(42),
		"name":            "HIST 301",
		"sis_section_id":  "2026-FALL-HIST-301-001",
	}
	if got := canvasSectionCodeFromRow(row); got != "2026-FALL-HIST-301-001" {
		t.Fatalf("canvasSectionCodeFromRow() = %q, want sis id", got)
	}
}

func TestCanvasSectionCodeFromRow_fallsBackToName(t *testing.T) {
	row := map[string]any{"id": float64(7), "name": "WMST 301"}
	if got := canvasSectionCodeFromRow(row); got != "WMST 301" {
		t.Fatalf("canvasSectionCodeFromRow() = %q, want name", got)
	}
}

func TestCanvasSectionCodeFromRow_fallsBackToCanvasID(t *testing.T) {
	row := map[string]any{"id": float64(99)}
	if got := canvasSectionCodeFromRow(row); got != "SEC-99" {
		t.Fatalf("canvasSectionCodeFromRow() = %q, want SEC-99", got)
	}
}

func TestCanvasEnrollmentSectionID_readsCourseSectionID(t *testing.T) {
	row := map[string]any{"course_section_id": float64(1234)}
	if got := canvasEnrollmentSectionID(row); got != 1234 {
		t.Fatalf("canvasEnrollmentSectionID() = %d, want 1234", got)
	}
}

func TestCanvasBuildSectionPendingRows_sortsByCode(t *testing.T) {
	rows := canvasBuildSectionPendingRows([]map[string]any{
		{"id": float64(2), "sis_section_id": "B"},
		{"id": float64(1), "sis_section_id": "A"},
	})
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].code != "A" || rows[1].code != "B" {
		t.Fatalf("codes = %q,%q, want A,B", rows[0].code, rows[1].code)
	}
}

func TestCanvasApplyCanvasDateOverrides_skipsBaseAndMissingSection(t *testing.T) {
	overrides := []map[string]any{
		{"base": true, "due_at": "2026-01-01T00:00:00Z"},
		{"course_section_id": float64(9), "due_at": "2026-02-01T00:00:00Z"},
		{"due_at": "2026-03-01T00:00:00Z"},
	}
	sectionMap := map[int64]uuid.UUID{1: uuid.New()}
	n, err := canvasApplyCanvasDateOverrides(context.TODO(), nil, overrides, sectionMap, uuid.New())
	if err != nil {
		t.Fatalf("canvasApplyCanvasDateOverrides: %v", err)
	}
	if n != 0 {
		t.Fatalf("imported = %d, want 0 without db", n)
	}
}