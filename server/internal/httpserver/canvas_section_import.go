package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/crosslisting"
)

type canvasSectionImportStats struct {
	SectionsCreated   int
	SectionsUpdated   int
	OverridesImported int
	CrossListLinked   bool
}

type canvasSectionPending struct {
	canvasID int64
	code     string
	name     *string
	meeting  json.RawMessage
	nonxlist int64
}

// canvasFetchCourseSections loads Canvas course sections for cross-listed and multi-section courses.
func canvasFetchCourseSections(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
) ([]map[string]any, error) {
	rows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/sections", canvasCourseID), nil)
	if err != nil {
		// Some tokens or course types omit sections; treat as no sections rather than failing import.
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return rows, nil
}

func canvasEnrollmentSectionID(row map[string]any) int64 {
	if row == nil {
		return 0
	}
	return int64At(row, "course_section_id")
}

func canvasSectionCodeFromRow(row map[string]any) string {
	if row == nil {
		return ""
	}
	if sis := strings.TrimSpace(strAt(row, "sis_section_id", "")); sis != "" {
		return sis
	}
	if name := strings.TrimSpace(strAt(row, "name", "")); name != "" {
		return name
	}
	if id := int64At(row, "id"); id > 0 {
		return fmt.Sprintf("SEC-%d", id)
	}
	return ""
}

func canvasSectionNameFromRow(row map[string]any) *string {
	if row == nil {
		return nil
	}
	name := strings.TrimSpace(strAt(row, "name", ""))
	if name == "" {
		return nil
	}
	return &name
}

func canvasSectionMeetingInfo(row map[string]any) json.RawMessage {
	meta := map[string]any{}
	if id := int64At(row, "id"); id > 0 {
		meta["canvas_section_id"] = id
	}
	if nxc := int64At(row, "nonxlist_course_id"); nxc > 0 {
		meta["canvas_nonxlist_course_id"] = nxc
	}
	if sis := strings.TrimSpace(strAt(row, "sis_section_id", "")); sis != "" {
		meta["canvas_sis_section_id"] = sis
	}
	raw, err := json.Marshal(meta)
	if err != nil || len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func canvasBuildSectionPendingRows(canvasSections []map[string]any) []canvasSectionPending {
	out := make([]canvasSectionPending, 0, len(canvasSections))
	for _, row := range canvasSections {
		canvasID := int64At(row, "id")
		code := canvasSectionCodeFromRow(row)
		if canvasID <= 0 || code == "" {
			continue
		}
		out = append(out, canvasSectionPending{
			canvasID: canvasID,
			code:     code,
			name:     canvasSectionNameFromRow(row),
			meeting:  canvasSectionMeetingInfo(row),
			nonxlist: int64At(row, "nonxlist_course_id"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].code < out[j].code })
	return out
}

func canvasImportCourseSections(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	orgID uuid.UUID,
	canvasSections []map[string]any,
) (map[int64]uuid.UUID, *canvasSectionImportStats, error) {
	stats := &canvasSectionImportStats{}
	pendingRows := canvasBuildSectionPendingRows(canvasSections)
	if len(pendingRows) == 0 {
		return nil, stats, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, errors.New("Failed to start import transaction.")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out := make(map[int64]uuid.UUID, len(pendingRows))
	for _, p := range pendingRows {
		var sectionID uuid.UUID
		err = tx.QueryRow(ctx, `
SELECT id FROM course.course_sections
WHERE course_id = $1 AND section_code = $2
`, courseID, p.code).Scan(&sectionID)
		switch {
		case err == nil:
			if _, err = tx.Exec(ctx, `
UPDATE course.course_sections
SET name = COALESCE($1, name), meeting_info = $2, status = 'active', updated_at = NOW()
WHERE id = $3 AND course_id = $4
`, p.name, p.meeting, sectionID, courseID); err != nil {
				return nil, nil, errors.New("Failed to update imported course section.")
			}
			stats.SectionsUpdated++
		case errors.Is(err, pgx.ErrNoRows):
			row := tx.QueryRow(ctx, `
INSERT INTO course.course_sections (course_id, section_code, name, meeting_info)
VALUES ($1, $2, $3, $4)
RETURNING id
`, courseID, p.code, p.name, p.meeting)
			if err = row.Scan(&sectionID); err != nil {
				return nil, nil, errors.New("Failed to create imported course section.")
			}
			stats.SectionsCreated++
		default:
			return nil, nil, errors.New("Failed to load existing course section.")
		}
		out[p.canvasID] = sectionID
	}

	if _, err = tx.Exec(ctx, `UPDATE course.courses SET sections_enabled = true, updated_at = NOW() WHERE id = $1`, courseID); err != nil {
		return nil, nil, errors.New("Failed to enable course sections.")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, errors.New("Something went wrong while saving imported sections.")
	}

	if err := canvasMaybeLinkCrossListedSections(ctx, pool, courseID, orgID, pendingRows, out, stats); err != nil {
		return out, stats, err
	}
	return out, stats, nil
}

func canvasMaybeLinkCrossListedSections(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, orgID uuid.UUID,
	sections []canvasSectionPending,
	sectionMap map[int64]uuid.UUID,
	stats *canvasSectionImportStats,
) error {
	if len(sections) < 2 {
		return nil
	}
	nonxlist := make(map[int64]struct{})
	for _, s := range sections {
		if s.nonxlist > 0 {
			nonxlist[s.nonxlist] = struct{}{}
		}
	}
	// Canvas cross-listed courses expose distinct nonxlist_course_id values per catalog shell.
	if len(nonxlist) < 2 {
		return nil
	}

	var existing int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.cross_list_groups WHERE course_id = $1`, courseID).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	primaryCanvasID := sections[0].canvasID
	primarySectionID, ok := sectionMap[primaryCanvasID]
	if !ok {
		return nil
	}

	groupName := "Canvas cross-listed sections"
	if _, err := crosslisting.CreateGroup(ctx, pool, orgID, courseID, primarySectionID, &groupName); err != nil {
		if errors.Is(err, crosslisting.ErrCourseHasGroup) {
			return nil
		}
		return errors.New("Failed to link cross-listed sections.")
	}
	for _, s := range sections[1:] {
		secID, ok := sectionMap[s.canvasID]
		if !ok {
			continue
		}
		if _, err := crosslisting.AddMember(ctx, pool, orgID, courseID, secID); err != nil {
			if errors.Is(err, crosslisting.ErrSectionBusy) || errors.Is(err, crosslisting.ErrCourseHasGroup) {
				continue
			}
			return errors.New("Failed to add cross-listed section.")
		}
	}
	stats.CrossListLinked = true
	return nil
}

func canvasImportSectionAssignmentOverrides(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	canvasAssignToItem map[int64]uuid.UUID,
	canvasQuizToItem map[int64]uuid.UUID,
	sectionMap map[int64]uuid.UUID,
) (int, error) {
	if len(sectionMap) == 0 || (len(canvasAssignToItem) == 0 && len(canvasQuizToItem) == 0) {
		return 0, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, errors.New("Failed to start import transaction.")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	imported := 0
	for canvasAssignID, itemID := range canvasAssignToItem {
		overrides, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d/assignments/%d/overrides", canvasCourseID, canvasAssignID), nil)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				continue
			}
			return imported, err
		}
		n, err := canvasApplyCanvasDateOverrides(ctx, tx, overrides, sectionMap, itemID)
		if err != nil {
			return imported, err
		}
		imported += n
	}

	for canvasQuizID, itemID := range canvasQuizToItem {
		obj, err := canvasGetObject(ctx, client, canvasBase, accessToken,
			fmt.Sprintf("courses/%d/quizzes/%d/date_details", canvasCourseID, canvasQuizID), nil)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				continue
			}
			return imported, err
		}
		n, err := canvasApplyCanvasDateOverrides(ctx, tx, arrAt(obj, "overrides"), sectionMap, itemID)
		if err != nil {
			return imported, err
		}
		imported += n
	}

	if imported == 0 {
		return 0, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return imported, errors.New("Something went wrong while saving section due date overrides.")
	}
	return imported, nil
}

func canvasApplyCanvasDateOverrides(
	ctx context.Context,
	tx pgx.Tx,
	overrides []map[string]any,
	sectionMap map[int64]uuid.UUID,
	structureItemID uuid.UUID,
) (int, error) {
	imported := 0
	for _, ov := range overrides {
		if boolAt(ov, "base", false) {
			continue
		}
		canvasSectionID := int64At(ov, "course_section_id")
		if canvasSectionID <= 0 {
			continue
		}
		sectionID, ok := sectionMap[canvasSectionID]
		if !ok {
			continue
		}
		dueAt := canvasTimeAt(ov, "due_at")
		unlockAt := canvasTimeAt(ov, "unlock_at")
		lockAt := canvasTimeAt(ov, "lock_at")
		if dueAt == nil && unlockAt == nil && lockAt == nil {
			continue
		}
		_, err := tx.Exec(ctx, `
INSERT INTO course.section_assignment_overrides (section_id, structure_item_id, due_at, available_from, available_until)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (section_id, structure_item_id) DO UPDATE SET
  due_at = EXCLUDED.due_at,
  available_from = EXCLUDED.available_from,
  available_until = EXCLUDED.available_until
`, sectionID, structureItemID, dueAt, unlockAt, lockAt)
		if err != nil {
			return imported, errors.New("Failed to save section due date override.")
		}
		imported++
	}
	return imported, nil
}

func canvasAssignSectionInstructor(
	ctx context.Context,
	tx pgx.Tx,
	sectionMap map[int64]uuid.UUID,
	canvasSectionID int64,
	instructorUserID uuid.UUID,
) error {
	if canvasSectionID <= 0 || instructorUserID == uuid.Nil {
		return nil
	}
	sectionID, ok := sectionMap[canvasSectionID]
	if !ok {
		return nil
	}
	_, err := tx.Exec(ctx, `
UPDATE course.course_sections
SET instructor_user_id = $1, updated_at = NOW()
WHERE id = $2 AND instructor_user_id IS NULL
`, instructorUserID, sectionID)
	return err
}