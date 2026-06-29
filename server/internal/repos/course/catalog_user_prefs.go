package course

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ValidCatalogViewTypes = map[string]struct{}{
	"cards": {}, "list": {}, "gallery": {}, "table": {}, "status": {},
}

var ValidKanbanColumnIDs = map[string]struct{}{
	"todo": {}, "in-progress": {}, "done": {}, "hidden": {},
}

var DefaultKanbanColumnLabels = map[string]string{
	"todo":        "Todo",
	"in-progress": "In progress",
	"done":        "Done",
	"hidden":      "Hidden",
}

type UserCatalogPrefs struct {
	ViewType             string            `json:"view"`
	KanbanColumnLabels   map[string]string `json:"kanbanColumnLabels"`
	HiddenColumnExpanded bool              `json:"hiddenColumnExpanded"`
}

type UserCatalogNickname struct {
	CourseID uuid.UUID
	Nickname string
}

type UserKanbanPlacement struct {
	CourseID  uuid.UUID
	ColumnID  string
	SortOrder int
}

type PinnedCourseSummary struct {
	ID                      uuid.UUID `json:"id"`
	CourseCode              string    `json:"courseCode"`
	Title                   string    `json:"title"`
	HeroImageURL            *string   `json:"heroImageUrl"`
	HeroImageObjectPosition *string   `json:"heroImageObjectPosition"`
	CatalogNickname         *string   `json:"catalogNickname,omitempty"`
}

const maxCatalogPins = 20
const maxCatalogPinsPerRow = 4

func defaultCatalogPrefs() UserCatalogPrefs {
	labels := make(map[string]string, len(DefaultKanbanColumnLabels))
	for k, v := range DefaultKanbanColumnLabels {
		labels[k] = v
	}
	return UserCatalogPrefs{
		ViewType:             "cards",
		KanbanColumnLabels:   labels,
		HiddenColumnExpanded: false,
	}
}

func normalizeKanbanColumnLabels(raw map[string]string) map[string]string {
	out := defaultCatalogPrefs().KanbanColumnLabels
	for k, v := range raw {
		if _, ok := ValidKanbanColumnIDs[k]; !ok {
			continue
		}
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || len(trimmed) > 80 {
			continue
		}
		out[k] = trimmed
	}
	return out
}

// GetUserCatalogPrefs returns catalog UI prefs for the user, or defaults when unset.
func GetUserCatalogPrefs(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (UserCatalogPrefs, error) {
	prefs := defaultCatalogPrefs()
	var labelsJSON []byte
	err := pool.QueryRow(ctx, `
SELECT view_type, kanban_column_labels, hidden_column_expanded
FROM course.user_course_catalog_prefs
WHERE user_id = $1
`, userID).Scan(&prefs.ViewType, &labelsJSON, &prefs.HiddenColumnExpanded)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return prefs, nil
		}
		return prefs, err
	}
	if _, ok := ValidCatalogViewTypes[prefs.ViewType]; !ok {
		prefs.ViewType = "cards"
	}
	var labels map[string]string
	if len(labelsJSON) > 0 {
		if err := json.Unmarshal(labelsJSON, &labels); err == nil {
			prefs.KanbanColumnLabels = normalizeKanbanColumnLabels(labels)
		}
	}
	return prefs, nil
}

// UpsertUserCatalogPrefs merges non-empty fields into stored prefs.
func UpsertUserCatalogPrefs(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, patch UserCatalogPrefs, hasView, hasLabels, hasHidden bool) (UserCatalogPrefs, error) {
	current, err := GetUserCatalogPrefs(ctx, pool, userID)
	if err != nil {
		return UserCatalogPrefs{}, err
	}
	if hasView {
		if _, ok := ValidCatalogViewTypes[patch.ViewType]; !ok {
			return UserCatalogPrefs{}, fmt.Errorf("invalid view type")
		}
		current.ViewType = patch.ViewType
	}
	if hasLabels {
		current.KanbanColumnLabels = normalizeKanbanColumnLabels(patch.KanbanColumnLabels)
	}
	if hasHidden {
		current.HiddenColumnExpanded = patch.HiddenColumnExpanded
	}
	labelsJSON, err := json.Marshal(current.KanbanColumnLabels)
	if err != nil {
		return UserCatalogPrefs{}, err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO course.user_course_catalog_prefs (user_id, view_type, kanban_column_labels, hidden_column_expanded, updated_at)
VALUES ($1, $2, $3::jsonb, $4, now())
ON CONFLICT (user_id) DO UPDATE SET
    view_type = EXCLUDED.view_type,
    kanban_column_labels = EXCLUDED.kanban_column_labels,
    hidden_column_expanded = EXCLUDED.hidden_column_expanded,
    updated_at = now()
`, userID, current.ViewType, labelsJSON, current.HiddenColumnExpanded)
	if err != nil {
		return UserCatalogPrefs{}, err
	}
	return current, nil
}

// ListUserCatalogNicknames returns nicknames keyed by course id string.
func ListUserCatalogNicknames(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (map[string]string, error) {
	rows, err := pool.Query(ctx, `
SELECT course_id, nickname
FROM course.user_course_catalog_nicknames
WHERE user_id = $1
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var courseID uuid.UUID
		var nickname string
		if err := rows.Scan(&courseID, &nickname); err != nil {
			return nil, err
		}
		out[courseID.String()] = nickname
	}
	return out, rows.Err()
}

// UpsertUserCatalogNickname sets or clears a nickname for one enrolled course.
func UpsertUserCatalogNickname(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, nickname *string) error {
	var enrolled bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.course_enrollments e
  WHERE e.user_id = $1 AND e.course_id = $2
    AND (e.active OR e.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
)
`, userID, courseID).Scan(&enrolled); err != nil {
		return err
	}
	if !enrolled {
		return fmt.Errorf("course is not in your catalog")
	}
	if nickname == nil || strings.TrimSpace(*nickname) == "" {
		_, err := pool.Exec(ctx, `
DELETE FROM course.user_course_catalog_nicknames
WHERE user_id = $1 AND course_id = $2
`, userID, courseID)
		return err
	}
	trimmed := strings.TrimSpace(*nickname)
	if len(trimmed) > 120 {
		return fmt.Errorf("nickname too long")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.user_course_catalog_nicknames (user_id, course_id, nickname, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (user_id, course_id) DO UPDATE SET
    nickname = EXCLUDED.nickname,
    updated_at = now()
`, userID, courseID, trimmed)
	return err
}

// ListUserKanbanPlacements returns manual kanban placements for the user.
func ListUserKanbanPlacements(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]UserKanbanPlacement, error) {
	rows, err := pool.Query(ctx, `
SELECT course_id, column_id, sort_order
FROM course.user_course_kanban_placement
WHERE user_id = $1
ORDER BY column_id ASC, sort_order ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserKanbanPlacement
	for rows.Next() {
		var p UserKanbanPlacement
		if err := rows.Scan(&p.CourseID, &p.ColumnID, &p.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ReplaceUserKanbanBoard replaces all kanban placements for enrolled courses provided in columns.
func ReplaceUserKanbanBoard(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, columns map[string][]uuid.UUID) error {
	for col := range columns {
		if _, ok := ValidKanbanColumnIDs[col]; !ok {
			return fmt.Errorf("invalid kanban column")
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	seen := map[uuid.UUID]struct{}{}
	var toInsert []UserKanbanPlacement
	for col, ids := range columns {
		for i, id := range ids {
			if _, dup := seen[id]; dup {
				return fmt.Errorf("duplicate course in kanban board")
			}
			seen[id] = struct{}{}
			toInsert = append(toInsert, UserKanbanPlacement{
				CourseID:  id,
				ColumnID:  col,
				SortOrder: i,
			})
		}
	}
	if len(toInsert) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM course.user_course_kanban_placement WHERE user_id = $1`, userID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	idList := make([]uuid.UUID, 0, len(toInsert))
	for id := range seen {
		idList = append(idList, id)
	}
	var enrolled int
	if err := tx.QueryRow(ctx, `
SELECT COUNT(DISTINCT e.course_id)
FROM course.course_enrollments e
WHERE e.user_id = $1 AND e.course_id = ANY($2::uuid[])
  AND (e.active OR e.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
`, userID, idList).Scan(&enrolled); err != nil {
		return err
	}
	if enrolled != len(idList) {
		return fmt.Errorf("one or more courses are not in your catalog")
	}

	if _, err := tx.Exec(ctx, `DELETE FROM course.user_course_kanban_placement WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, p := range toInsert {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.user_course_kanban_placement (user_id, course_id, column_id, sort_order, updated_at)
VALUES ($1, $2, $3, $4, now())
`, userID, p.CourseID, p.ColumnID, p.SortOrder); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ReplaceUserCatalogOrder replaces the user's catalog sort order.
func ReplaceUserCatalogOrder(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) error {
	if len(courseIDs) == 0 {
		_, err := pool.Exec(ctx, `DELETE FROM course.user_course_catalog_order WHERE user_id = $1`, userID)
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var enrolled int
	if err := tx.QueryRow(ctx, `
SELECT COUNT(DISTINCT e.course_id)
FROM course.course_enrollments e
WHERE e.user_id = $1 AND e.course_id = ANY($2::uuid[])
  AND (e.active OR e.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
`, userID, courseIDs).Scan(&enrolled); err != nil {
		return err
	}
	if enrolled != len(courseIDs) {
		return fmt.Errorf("one or more courses are not in your catalog")
	}
	seen := map[uuid.UUID]struct{}{}
	for _, id := range courseIDs {
		if _, dup := seen[id]; dup {
			return fmt.Errorf("duplicate course in catalog order")
		}
		seen[id] = struct{}{}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM course.user_course_catalog_order WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for i, id := range courseIDs {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.user_course_catalog_order (user_id, course_id, sort_order)
VALUES ($1, $2, $3)
`, userID, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListUserCatalogPinIDs returns pinned course ids in display order.
func ListUserCatalogPinIDs(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT course_id
FROM course.user_course_catalog_pins
WHERE user_id = $1
ORDER BY row_index ASC, sort_order ASC, updated_at ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var courseID uuid.UUID
		if err := rows.Scan(&courseID); err != nil {
			return nil, err
		}
		out = append(out, courseID)
	}
	return out, rows.Err()
}

func userIsEnrolledInCatalog(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	var enrolled bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.course_enrollments e
  WHERE e.user_id = $1 AND e.course_id = $2
    AND (e.active OR e.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
)
`, userID, courseID).Scan(&enrolled)
	return enrolled, err
}

// SetUserCatalogPin pins or unpins one enrolled course for the user.
func SetUserCatalogPin(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, pinned bool) error {
	if !pinned {
		_, err := pool.Exec(ctx, `
DELETE FROM course.user_course_catalog_pins
WHERE user_id = $1 AND course_id = $2
`, userID, courseID)
		return err
	}
	enrolled, err := userIsEnrolledInCatalog(ctx, pool, userID, courseID)
	if err != nil {
		return err
	}
	if !enrolled {
		return fmt.Errorf("course is not in your catalog")
	}
	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.user_course_catalog_pins WHERE user_id = $1
`, userID).Scan(&count); err != nil {
		return err
	}
	if count >= maxCatalogPins {
		return fmt.Errorf("pin limit reached")
	}
	var already bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.user_course_catalog_pins WHERE user_id = $1 AND course_id = $2
)
`, userID, courseID).Scan(&already); err != nil {
		return err
	}
	if already {
		return nil
	}
	var rowIndex, sortOrder int
	if err := pool.QueryRow(ctx, `
SELECT COALESCE(MAX(row_index), 0), COUNT(*)::int
FROM course.user_course_catalog_pins
WHERE user_id = $1 AND row_index = (SELECT COALESCE(MAX(row_index), 0) FROM course.user_course_catalog_pins WHERE user_id = $1)
`, userID).Scan(&rowIndex, &sortOrder); err != nil {
		return err
	}
	if sortOrder >= maxCatalogPinsPerRow {
		rowIndex++
		sortOrder = 0
	}
	_, err = pool.Exec(ctx, `
INSERT INTO course.user_course_catalog_pins (user_id, course_id, row_index, sort_order, updated_at)
VALUES ($1, $2, $3, $4, now())
`, userID, courseID, rowIndex, sortOrder)
	return err
}

// ReplaceUserCatalogPinLayout replaces the sidebar pin grid for the user.
// Each inner slice is one row (at most maxCatalogPinsPerRow course ids).
func ReplaceUserCatalogPinLayout(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, rows [][]uuid.UUID) error {
	seen := map[uuid.UUID]struct{}{}
	var placements []struct {
		courseID  uuid.UUID
		rowIndex  int
		sortOrder int
	}
	for rowIndex, row := range rows {
		if len(row) > maxCatalogPinsPerRow {
			return fmt.Errorf("pin row exceeds limit")
		}
		for sortOrder, id := range row {
			if _, dup := seen[id]; dup {
				return fmt.Errorf("duplicate course in pin layout")
			}
			seen[id] = struct{}{}
			placements = append(placements, struct {
				courseID  uuid.UUID
				rowIndex  int
				sortOrder int
			}{courseID: id, rowIndex: rowIndex, sortOrder: sortOrder})
		}
	}
	if len(placements) == 0 {
		return nil
	}

	idList := make([]uuid.UUID, 0, len(placements))
	for id := range seen {
		idList = append(idList, id)
	}
	var pinned int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM course.user_course_catalog_pins
WHERE user_id = $1 AND course_id = ANY($2::uuid[])
`, userID, idList).Scan(&pinned); err != nil {
		return err
	}
	if pinned != len(idList) {
		return fmt.Errorf("pin layout includes unpinned courses")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, p := range placements {
		if _, err := tx.Exec(ctx, `
UPDATE course.user_course_catalog_pins
SET row_index = $3, sort_order = $4, updated_at = now()
WHERE user_id = $1 AND course_id = $2
`, userID, p.courseID, p.rowIndex, p.sortOrder); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListUserPinnedCourseRows returns pinned courses grouped by sidebar row.
func ListUserPinnedCourseRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([][]PinnedCourseSummary, error) {
	rows, err := pool.Query(ctx, `
SELECT
    c.id,
    c.course_code,
    c.title,
    c.hero_image_url,
    c.hero_image_object_position,
    n.nickname,
    p.row_index
FROM course.user_course_catalog_pins p
JOIN course.courses c ON c.id = p.course_id
JOIN course.course_enrollments e ON e.course_id = c.id AND e.user_id = p.user_id
LEFT JOIN course.user_course_catalog_nicknames n ON n.user_id = p.user_id AND n.course_id = c.id
WHERE p.user_id = $1
  AND (e.active OR e.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
  AND NOT c.archived
ORDER BY p.row_index ASC, p.sort_order ASC, p.updated_at ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rowByIndex := map[int][]PinnedCourseSummary{}
	var rowOrder []int
	for rows.Next() {
		var item PinnedCourseSummary
		var rowIndex int
		if err := rows.Scan(
			&item.ID,
			&item.CourseCode,
			&item.Title,
			&item.HeroImageURL,
			&item.HeroImageObjectPosition,
			&item.CatalogNickname,
			&rowIndex,
		); err != nil {
			return nil, err
		}
		if _, ok := rowByIndex[rowIndex]; !ok {
			rowOrder = append(rowOrder, rowIndex)
		}
		rowByIndex[rowIndex] = append(rowByIndex[rowIndex], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([][]PinnedCourseSummary, 0, len(rowOrder))
	for _, rowIndex := range rowOrder {
		out = append(out, rowByIndex[rowIndex])
	}
	return out, nil
}

// ListUserPinnedCourseSummaries returns pinned enrolled courses for sidebar shortcuts.
func ListUserPinnedCourseSummaries(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]PinnedCourseSummary, error) {
	rows, err := pool.Query(ctx, `
SELECT
    c.id,
    c.course_code,
    c.title,
    c.hero_image_url,
    c.hero_image_object_position,
    n.nickname
FROM course.user_course_catalog_pins p
JOIN course.courses c ON c.id = p.course_id
JOIN course.course_enrollments e ON e.course_id = c.id AND e.user_id = p.user_id
LEFT JOIN course.user_course_catalog_nicknames n ON n.user_id = p.user_id AND n.course_id = c.id
WHERE p.user_id = $1
  AND (e.active OR e.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
  AND NOT c.archived
ORDER BY p.row_index ASC, p.sort_order ASC, p.updated_at ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PinnedCourseSummary
	for rows.Next() {
		var item PinnedCourseSummary
		if err := rows.Scan(
			&item.ID,
			&item.CourseCode,
			&item.Title,
			&item.HeroImageURL,
			&item.HeroImageObjectPosition,
			&item.CatalogNickname,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// AttachUserCatalogMeta merges nicknames, kanban placement, and pin state onto listed courses.
func AttachUserCatalogMeta(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courses []CoursePublic) error {
	if len(courses) == 0 {
		return nil
	}
	nicknames, err := ListUserCatalogNicknames(ctx, pool, userID)
	if err != nil {
		return err
	}
	placements, err := ListUserKanbanPlacements(ctx, pool, userID)
	if err != nil {
		return err
	}
	pinIDs, err := ListUserCatalogPinIDs(ctx, pool, userID)
	if err != nil {
		return err
	}
	placementByCourse := map[string]UserKanbanPlacement{}
	for _, p := range placements {
		placementByCourse[p.CourseID.String()] = p
	}
	pinned := map[string]struct{}{}
	for _, id := range pinIDs {
		pinned[id.String()] = struct{}{}
	}
	for i := range courses {
		if nick, ok := nicknames[courses[i].ID]; ok {
			n := nick
			courses[i].CatalogNickname = &n
		}
		if p, ok := placementByCourse[courses[i].ID]; ok {
			col := p.ColumnID
			sortOrder := p.SortOrder
			courses[i].KanbanColumnID = &col
			courses[i].KanbanSortOrder = &sortOrder
		}
		if _, ok := pinned[courses[i].ID]; ok {
			courses[i].CatalogPinned = true
		}
	}
	return nil
}
