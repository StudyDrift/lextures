package coursecopy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// Options configures a one-way copy from an existing course into a newly created empty target course.
type Options struct {
	SourceCourseID   uuid.UUID
	TargetCourseID   uuid.UUID
	SourceCourseCode string
	TargetCourseCode string
	Include          Include
	FilesRoot        string
	ActorUserID      uuid.UUID
}

// CopyFromCourse copies selected content from source into target. Target must be empty (no structure rows).
func CopyFromCourse(ctx context.Context, pool *pgxpool.Pool, opts Options) error {
	include := opts.Include.WithDefaults()

	// Read source structure (and quiz adaptive refs) before opening a transaction — pool queries
	// while a tx holds the same connection return "conn busy" in pgx.
	var structureRows []coursestructure.ItemRow
	quizAdaptiveSources := map[uuid.UUID][]uuid.UUID{}
	if include.wantsStructure() {
		rows, err := coursestructure.ListForCourse(ctx, pool, opts.SourceCourseID)
		if err != nil {
			return err
		}
		structureRows = rows
		if include.Quizzes {
			for _, r := range coursestructure.OrderRows(rows) {
				if r.Kind != "quiz" || !include.shouldCopyKind(r.Kind) {
					continue
				}
				qrow, err := coursemodulequizzes.GetForCourseItem(ctx, pool, opts.SourceCourseID, r.ID)
				if err != nil {
					return err
				}
				if qrow != nil && len(qrow.AdaptiveSourceItemIDs) > 0 {
					quizAdaptiveSources[r.ID] = qrow.AdaptiveSourceItemIDs
				}
			}
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if include.Settings {
		if err := copySettings(ctx, tx, opts.SourceCourseID, opts.TargetCourseID); err != nil {
			return err
		}
	}

	groupMap, err := copyAssignmentGroups(ctx, tx, opts.SourceCourseID, opts.TargetCourseID, include)
	if err != nil {
		return err
	}

	itemMap, err := copyStructure(ctx, tx, opts, include, groupMap, structureRows, quizAdaptiveSources)
	if err != nil {
		return err
	}

	if include.Enrollments {
		if err := copyEnrollments(ctx, tx, opts.SourceCourseID, opts.TargetCourseID, opts.TargetCourseCode); err != nil {
			return err
		}
	}

	if include.Grades && len(itemMap) > 0 {
		if err := copyGrades(ctx, tx, opts.SourceCourseID, opts.TargetCourseID, itemMap); err != nil {
			return err
		}
	}

	if include.Files {
		if err := copyCourseFiles(ctx, tx, opts); err != nil {
			return err
		}
		if err := copyFileManager(ctx, tx, opts); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func copySettings(ctx context.Context, tx pgx.Tx, sourceID, targetID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		UPDATE course.courses AS tgt
		SET
			description = src.description,
			hero_image_url = src.hero_image_url,
			hero_image_object_position = src.hero_image_object_position,
			starts_at = src.starts_at,
			ends_at = src.ends_at,
			visible_from = src.visible_from,
			hidden_at = src.hidden_at,
			schedule_mode = src.schedule_mode,
			relative_end_after = src.relative_end_after,
			relative_hidden_after = src.relative_hidden_after,
			relative_schedule_anchor_at = src.relative_schedule_anchor_at,
			published = src.published,
			markdown_theme_preset = src.markdown_theme_preset,
			markdown_theme_custom = src.markdown_theme_custom,
			notebook_enabled = src.notebook_enabled,
			feed_enabled = src.feed_enabled,
			calendar_enabled = src.calendar_enabled,
			question_bank_enabled = src.question_bank_enabled,
			lockdown_mode_enabled = src.lockdown_mode_enabled,
			standards_alignment_enabled = src.standards_alignment_enabled,
			adaptive_paths_enabled = src.adaptive_paths_enabled,
			srs_enabled = src.srs_enabled,
			diagnostic_assessments_enabled = src.diagnostic_assessments_enabled,
			hint_scaffolding_enabled = src.hint_scaffolding_enabled,
			misconception_detection_enabled = src.misconception_detection_enabled,
			files_enabled = src.files_enabled,
			grading_scale = src.grading_scale,
			sbg_enabled = src.sbg_enabled,
			sbg_proficiency_scale_json = src.sbg_proficiency_scale_json,
			sbg_aggregation_rule = src.sbg_aggregation_rule,
			course_home_landing = src.course_home_landing,
			course_timezone = src.course_timezone,
			updated_at = NOW()
		FROM course.courses AS src
		WHERE tgt.id = $1 AND src.id = $2
	`, targetID, sourceID); err != nil {
		return err
	}

	var sections []byte
	var require bool
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(cs.sections, '[]'::jsonb), COALESCE(cs.require_syllabus_acceptance, false)
		FROM course.course_syllabus cs
		WHERE cs.course_id = $1
	`, sourceID).Scan(&sections, &require)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if err == nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.course_syllabus (course_id, sections, require_syllabus_acceptance, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (course_id) DO UPDATE SET
				sections = EXCLUDED.sections,
				require_syllabus_acceptance = EXCLUDED.require_syllabus_acceptance,
				updated_at = NOW()
		`, targetID, sections, require); err != nil {
			return err
		}
	}
	return nil
}

func copyAssignmentGroups(ctx context.Context, tx pgx.Tx, sourceID, targetID uuid.UUID, include Include) (map[uuid.UUID]uuid.UUID, error) {
	groupMap := make(map[uuid.UUID]uuid.UUID)
	if !include.Settings && !include.Assignments {
		return groupMap, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id, sort_order, name, weight_percent, drop_lowest, drop_highest, replace_lowest_with_final
		FROM course.assignment_groups
		WHERE course_id = $1
		ORDER BY sort_order ASC, name ASC
	`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			oldID   uuid.UUID
			sort    int
			name    string
			weight  float64
			dropLo  int
			dropHi  int
			replace bool
		)
		if err := rows.Scan(&oldID, &sort, &name, &weight, &dropLo, &dropHi, &replace); err != nil {
			return nil, err
		}
		newID := uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.assignment_groups (
				id, course_id, sort_order, name, weight_percent, drop_lowest, drop_highest, replace_lowest_with_final
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, newID, targetID, sort, name, weight, dropLo, dropHi, replace); err != nil {
			return nil, err
		}
		groupMap[oldID] = newID
	}
	return groupMap, rows.Err()
}

func copyStructure(
	ctx context.Context,
	tx pgx.Tx,
	opts Options,
	include Include,
	groupMap map[uuid.UUID]uuid.UUID,
	structureRows []coursestructure.ItemRow,
	quizAdaptiveSources map[uuid.UUID][]uuid.UUID,
) (map[uuid.UUID]uuid.UUID, error) {
	itemMap := make(map[uuid.UUID]uuid.UUID)
	if !include.wantsStructure() {
		return itemMap, nil
	}

	ordered := coursestructure.OrderRows(structureRows)

	for _, r := range ordered {
		if !include.shouldCopyKind(r.Kind) {
			continue
		}
		var newParent *uuid.UUID
		if r.ParentID != nil {
			p, ok := itemMap[*r.ParentID]
			if !ok {
				return nil, fmt.Errorf("coursecopy: broken parent chain for item %s", r.ID)
			}
			newParent = &p
		}
		var newGroupID *uuid.UUID
		if r.AssignmentGroupID != nil {
			if mapped, ok := groupMap[*r.AssignmentGroupID]; ok {
				newGroupID = &mapped
			}
		}
		newID := uuid.New()
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.course_structure_items (
				id, course_id, sort_order, kind, title, parent_id,
				published, visible_from, archived, due_at, assignment_group_id,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				NOW(), NOW()
			)
		`, newID, opts.TargetCourseID, r.SortOrder, r.Kind, r.Title, newParent,
			r.Published, r.VisibleFrom, r.Archived, r.DueAt, newGroupID); err != nil {
			return nil, err
		}
		itemMap[r.ID] = newID
	}

	for _, r := range ordered {
		if !include.shouldCopyKind(r.Kind) {
			continue
		}
		dst := itemMap[r.ID]
		if err := copyItemExtensions(ctx, tx, opts.TargetCourseID, r.Kind, r.ID, dst, itemMap, quizAdaptiveSources); err != nil {
			return nil, err
		}
	}
	return itemMap, nil
}

func copyItemExtensions(
	ctx context.Context,
	tx pgx.Tx,
	targetCourseID uuid.UUID,
	kind string,
	srcItemID, dstItemID uuid.UUID,
	itemMap map[uuid.UUID]uuid.UUID,
	quizAdaptiveSources map[uuid.UUID][]uuid.UUID,
) error {
	switch kind {
	case "module", "heading":
		return nil
	case "content_page":
		_, err := tx.Exec(ctx, `
			INSERT INTO course.module_content_pages (structure_item_id, markdown, updated_at)
			SELECT $1::uuid, m.markdown, NOW()
			FROM course.module_content_pages m WHERE m.structure_item_id = $2
		`, dstItemID, srcItemID)
		return err
	case "assignment":
		_, err := tx.Exec(ctx, copyAssignmentSQL, dstItemID, srcItemID)
		return err
	case "quiz":
		if _, err := tx.Exec(ctx, copyQuizSQL, dstItemID, srcItemID); err != nil {
			return err
		}
		if srcIDs, ok := quizAdaptiveSources[srcItemID]; ok && len(srcIDs) > 0 {
			rem := remapAdaptiveSources(srcIDs, itemMap)
			b, err := json.Marshal(rem)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `
				UPDATE course.module_quizzes SET adaptive_source_item_ids = $2::jsonb, updated_at = NOW()
				WHERE structure_item_id = $1
			`, dstItemID, b)
			return err
		}
		return nil
	case "external_link":
		_, err := tx.Exec(ctx, `
			INSERT INTO course.module_external_links (structure_item_id, url, updated_at)
			SELECT $1::uuid, m.url, NOW()
			FROM course.module_external_links m WHERE m.structure_item_id = $2
		`, dstItemID, srcItemID)
		return err
	case "survey":
		_, err := tx.Exec(ctx, `
			INSERT INTO course.module_surveys (
				structure_item_id, description, anonymity_mode, opens_at, closes_at, questions_json, updated_at
			)
			SELECT $1::uuid, m.description, m.anonymity_mode, m.opens_at, m.closes_at, m.questions_json, NOW()
			FROM course.module_surveys m WHERE m.structure_item_id = $2
		`, dstItemID, srcItemID)
		return err
	case "lti_link":
		_, err := tx.Exec(ctx, `
			INSERT INTO course.lti_resource_links (
				id, course_id, structure_item_id, external_tool_id, resource_link_id, title, custom_params, line_item_url, created_at
			)
			SELECT gen_random_uuid(), $3::uuid, $1::uuid, m.external_tool_id, m.resource_link_id, m.title, m.custom_params, m.line_item_url, NOW()
			FROM course.lti_resource_links m
			WHERE m.structure_item_id = $2
		`, dstItemID, srcItemID, targetCourseID)
		return err
	case "vibe_activity":
		_, err := tx.Exec(ctx, `
			INSERT INTO course.module_vibe_activities (structure_item_id, html_content, updated_at)
			SELECT $1::uuid, m.html_content, NOW()
			FROM course.module_vibe_activities m WHERE m.structure_item_id = $2
		`, dstItemID, srcItemID)
		return err
	default:
		return fmt.Errorf("coursecopy: unsupported structure kind %q", kind)
	}
}

func remapAdaptiveSources(src []uuid.UUID, idMap map[uuid.UUID]uuid.UUID) []uuid.UUID {
	if len(src) == 0 {
		return nil
	}
	out := make([]uuid.UUID, 0, len(src))
	for _, x := range src {
		if u, ok := idMap[x]; ok {
			out = append(out, u)
		}
	}
	return out
}

func copyEnrollments(ctx context.Context, tx pgx.Tx, sourceID, targetID uuid.UUID, targetCourseCode string) error {
	rows, err := tx.Query(ctx, `
		SELECT user_id, role
		FROM course.course_enrollments
		WHERE course_id = $1 AND active = true
	`, sourceID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type pair struct {
		userID uuid.UUID
		role   string
	}
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.userID, &p.role); err != nil {
			return err
		}
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range pairs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.course_enrollments (course_id, user_id, role, active)
			SELECT $1, $2, $3, true
			WHERE NOT EXISTS (
				SELECT 1 FROM course.course_enrollments
				WHERE course_id = $1 AND user_id = $2 AND role = $3
			)
		`, targetID, p.userID, p.role); err != nil {
			return err
		}
		if p.role == "teacher" {
			if err := course.SeedTeacherCourseGrants(ctx, tx, p.userID, targetID, targetCourseCode); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyGrades(ctx context.Context, tx pgx.Tx, sourceID, targetID uuid.UUID, itemMap map[uuid.UUID]uuid.UUID) error {
	for srcItem, dstItem := range itemMap {
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.course_grades (
				course_id, student_user_id, module_item_id, points_earned, updated_at, posted_at
			)
			SELECT $1, g.student_user_id, $2::uuid, g.points_earned, NOW(), g.posted_at
			FROM course.course_grades g
			WHERE g.course_id = $3 AND g.module_item_id = $4::uuid
			ON CONFLICT (course_id, student_user_id, module_item_id) DO UPDATE SET
				points_earned = EXCLUDED.points_earned,
				updated_at = NOW(),
				posted_at = COALESCE(course.course_grades.posted_at, EXCLUDED.posted_at)
		`, targetID, dstItem, sourceID, srcItem); err != nil {
			return err
		}
	}
	return nil
}

func copyCourseFiles(ctx context.Context, tx pgx.Tx, opts Options) error {
	root := filepath.Clean(strings.TrimSpace(opts.FilesRoot))
	if root == "" {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT storage_key, original_filename, mime_type, byte_size, uploaded_by
		FROM course.course_files
		WHERE course_id = $1
	`, opts.SourceCourseID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			storageKey       string
			originalFilename string
			mimeType         string
			byteSize         int64
			uploadedBy       *uuid.UUID
		)
		if err := rows.Scan(&storageKey, &originalFilename, &mimeType, &byteSize, &uploadedBy); err != nil {
			return err
		}
		newKey := uuid.New().String()
		srcPath := coursefiles.BlobDiskPath(root, opts.SourceCourseCode, storageKey)
		dstPath := coursefiles.BlobDiskPath(root, opts.TargetCourseCode, newKey)
		if err := copyBlobFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy course file %q: %w", originalFilename, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.course_files (course_id, storage_key, original_filename, mime_type, byte_size, uploaded_by)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, opts.TargetCourseID, newKey, originalFilename, mimeType, byteSize, uploadedBy); err != nil {
			return err
		}
	}
	return rows.Err()
}

func copyFileManager(ctx context.Context, tx pgx.Tx, opts Options) error {
	root := filepath.Clean(strings.TrimSpace(opts.FilesRoot))
	folderMap := make(map[uuid.UUID]uuid.UUID)

	folderRows, err := tx.Query(ctx, `
		SELECT id, parent_id, name, created_by
		FROM course.file_folders
		WHERE course_id = $1
		ORDER BY created_at ASC
	`, opts.SourceCourseID)
	if err != nil {
		return err
	}
	defer folderRows.Close()
	type folderRow struct {
		id        uuid.UUID
		parentID  *uuid.UUID
		name      string
		createdBy *uuid.UUID
	}
	var folders []folderRow
	for folderRows.Next() {
		var f folderRow
		if err := folderRows.Scan(&f.id, &f.parentID, &f.name, &f.createdBy); err != nil {
			return err
		}
		folders = append(folders, f)
	}
	if err := folderRows.Err(); err != nil {
		return err
	}
	for _, f := range folders {
		newID := uuid.New()
		var newParent *uuid.UUID
		if f.parentID != nil {
			if p, ok := folderMap[*f.parentID]; ok {
				newParent = &p
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.file_folders (id, course_id, parent_id, name, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, newID, opts.TargetCourseID, newParent, f.name, f.createdBy); err != nil {
			return err
		}
		folderMap[f.id] = newID
	}

	if root == "" {
		return nil
	}
	itemRows, err := tx.Query(ctx, `
		SELECT folder_id, storage_key, original_filename, display_name, mime_type, byte_size, uploaded_by, canvas_file_id
		FROM course.file_items
		WHERE course_id = $1
	`, opts.SourceCourseID)
	if err != nil {
		return err
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var (
			folderID         *uuid.UUID
			storageKey       string
			originalFilename string
			displayName      string
			mimeType         string
			byteSize         int64
			uploadedBy       *uuid.UUID
			canvasFileID     *int64
		)
		if err := itemRows.Scan(&folderID, &storageKey, &originalFilename, &displayName, &mimeType, &byteSize, &uploadedBy, &canvasFileID); err != nil {
			return err
		}
		newKey := "fm-" + uuid.New().String()
		srcPath := coursefiles.BlobDiskPath(root, opts.SourceCourseCode, storageKey)
		dstPath := coursefiles.BlobDiskPath(root, opts.TargetCourseCode, newKey)
		if err := copyBlobFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy file manager item %q: %w", displayName, err)
		}
		var newFolderID *uuid.UUID
		if folderID != nil {
			if mapped, ok := folderMap[*folderID]; ok {
				newFolderID = &mapped
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.file_items (
				course_id, folder_id, storage_key, original_filename, display_name,
				mime_type, byte_size, uploaded_by, canvas_file_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`, opts.TargetCourseID, newFolderID, newKey, originalFilename, displayName, mimeType, byteSize, uploadedBy, canvasFileID); err != nil {
			return err
		}
	}
	return itemRows.Err()
}

func copyBlobFile(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

