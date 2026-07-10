package marketplacecourses

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	mcrepo "github.com/lextures/lextures/server/internal/repos/marketplacecourses"
)

// ContentSyncReport summarizes a content sync run.
type ContentSyncReport struct {
	Skipped     bool
	Modules     int
	Pages       int
	Assignments int
	Quizzes     int
	Archived    int
	Updated     int
	Unchanged   int
}

// SyncContent upserts curriculum fixtures into an official marketplace course.
func SyncContent(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, spec *CourseSpec) (ContentSyncReport, error) {
	started := time.Now()
	report := ContentSyncReport{}
	if spec == nil {
		return report, fmt.Errorf("course spec is nil")
	}
	courseSlug := spec.Manifest.CatalogSlug
	if courseSlug == "" {
		courseSlug = spec.Manifest.DirSlug
	}

	if err := ValidateCourseSpec(spec); err != nil {
		recordContentSync(courseSlug, "error", started)
		return report, err
	}

	desired := make(map[string]struct{})
	itemVersions := make(map[string]int)
	for _, mod := range spec.Modules {
		desired[mod.Meta.Slug] = struct{}{}
		itemVersions[mod.Meta.Slug] = spec.Manifest.ContentVersion
		for _, p := range mod.Pages {
			desired[p.Slug] = struct{}{}
			itemVersions[p.Slug] = p.ContentVer
		}
		for _, a := range mod.Assignments {
			desired[a.Slug] = struct{}{}
			itemVersions[a.Slug] = a.ContentVer
		}
		for _, q := range mod.Quizzes {
			desired[q.Slug] = struct{}{}
			itemVersions[q.Slug] = q.ContentVer
		}
	}

	upToDate, err := curriculumUpToDate(ctx, tx, courseID, courseSlug, desired, itemVersions)
	if err != nil {
		recordContentSync(courseSlug, "error", started)
		return report, err
	}
	syllabusOK, err := syllabusUpToDate(ctx, tx, courseID, spec.Syllabus)
	if err != nil {
		recordContentSync(courseSlug, "error", started)
		return report, err
	}
	if upToDate && syllabusOK {
		report.Skipped = true
		recordContentSync(courseSlug, "noop", started)
		return report, nil
	}

	for _, mod := range spec.Modules {
		moduleID, updated, err := upsertModule(ctx, tx, courseID, courseSlug, mod.Meta, itemVersions[mod.Meta.Slug])
		if err != nil {
			recordContentSync(courseSlug, "error", started)
			return report, fmt.Errorf("module %s: %w", mod.Meta.Slug, err)
		}
		report.Modules++
		if updated {
			report.Updated++
		} else {
			report.Unchanged++
		}

		for _, quiz := range mod.Quizzes {
			updated, err := upsertQuiz(ctx, tx, courseID, courseSlug, moduleID, quiz)
			if err != nil {
				recordContentSync(courseSlug, "error", started)
				return report, fmt.Errorf("quiz %s: %w", quiz.Slug, err)
			}
			report.Quizzes++
			if updated {
				report.Updated++
			} else {
				report.Unchanged++
			}
		}
		for _, page := range mod.Pages {
			updated, err := upsertContentPage(ctx, tx, courseID, courseSlug, moduleID, page)
			if err != nil {
				recordContentSync(courseSlug, "error", started)
				return report, fmt.Errorf("page %s: %w", page.Slug, err)
			}
			report.Pages++
			if updated {
				report.Updated++
			} else {
				report.Unchanged++
			}
		}
		for _, assign := range mod.Assignments {
			updated, err := upsertAssignment(ctx, tx, courseID, courseSlug, moduleID, assign)
			if err != nil {
				recordContentSync(courseSlug, "error", started)
				return report, fmt.Errorf("assignment %s: %w", assign.Slug, err)
			}
			report.Assignments++
			if updated {
				report.Updated++
			} else {
				report.Unchanged++
			}
		}
	}

	archived, err := archiveRemovedItems(ctx, tx, courseID, courseSlug, desired)
	if err != nil {
		recordContentSync(courseSlug, "error", started)
		return report, err
	}
	report.Archived = archived

	if err := syncSyllabus(ctx, tx, courseID, spec.Syllabus); err != nil {
		recordContentSync(courseSlug, "error", started)
		return report, fmt.Errorf("syllabus: %w", err)
	}

	recordContentSync(courseSlug, "success", started)
	return report, nil
}

func curriculumUpToDate(
	ctx context.Context,
	tx pgx.Tx,
	courseID uuid.UUID,
	courseSlug string,
	desired map[string]struct{},
	itemVersions map[string]int,
) (bool, error) {
	for slug := range desired {
		itemID, err := mcrepo.LookupContentItem(ctx, tx, courseSlug, slug)
		if err != nil {
			return false, err
		}
		if itemID == nil {
			return false, nil
		}
		var version int
		var archived bool
		err = tx.QueryRow(ctx, `
SELECT mci.content_version, csi.archived
FROM settings.marketplace_course_items mci
INNER JOIN course.course_structure_items csi ON csi.id = mci.structure_item_id
WHERE mci.course_slug = $1 AND mci.slug = $2 AND csi.course_id = $3
`, courseSlug, slug, courseID).Scan(&version, &archived)
		if err != nil {
			return false, err
		}
		want := itemVersions[slug]
		if archived || version != want {
			return false, nil
		}
	}

	rows, err := mcrepo.ListContentItems(ctx, tx, courseSlug)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if _, keep := desired[row.Slug]; keep {
			continue
		}
		var archived bool
		err := tx.QueryRow(ctx, `
SELECT archived FROM course.course_structure_items WHERE id = $1 AND course_id = $2
`, row.StructureItemID, courseID).Scan(&archived)
		if err != nil {
			return false, err
		}
		if !archived {
			return false, nil
		}
	}
	return true, nil
}

func itemNeedsSync(ctx context.Context, tx pgx.Tx, courseSlug, slug string, wantVersion int) (existing *uuid.UUID, needsBody bool, err error) {
	existing, err = mcrepo.LookupContentItem(ctx, tx, courseSlug, slug)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, true, nil
	}
	stored, ok, err := mcrepo.ItemContentVersion(ctx, tx, courseSlug, slug)
	if err != nil {
		return nil, false, err
	}
	if !ok || stored != wantVersion {
		return existing, true, nil
	}
	return existing, false, nil
}

func upsertModule(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, courseSlug string, meta ModuleMeta, contentVersion int) (uuid.UUID, bool, error) {
	existing, needsBody, err := itemNeedsSync(ctx, tx, courseSlug, meta.Slug, contentVersion)
	if err != nil {
		return uuid.Nil, false, err
	}
	if existing != nil {
		if _, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $2, sort_order = $3, published = TRUE, archived = FALSE, updated_at = NOW()
WHERE id = $1 AND course_id = $4 AND kind = 'module' AND parent_id IS NULL
`, *existing, meta.Title, meta.SortOrder, courseID); err != nil {
			return uuid.Nil, false, err
		}
		if err := mcrepo.UpsertContentItem(ctx, tx, courseSlug, meta.Slug, *existing, contentVersion, nil); err != nil {
			return uuid.Nil, false, err
		}
		return *existing, needsBody, nil
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived)
VALUES ($1, $2, 'module', $3, NULL, TRUE, FALSE)
RETURNING id
`, courseID, meta.SortOrder, meta.Title).Scan(&id)
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, mcrepo.UpsertContentItem(ctx, tx, courseSlug, meta.Slug, id, contentVersion, nil)
}

func upsertContentPage(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, courseSlug string, moduleID uuid.UUID, page PageFixture) (bool, error) {
	existing, needsBody, err := itemNeedsSync(ctx, tx, courseSlug, page.Slug, page.ContentVer)
	if err != nil {
		return false, err
	}
	var itemID uuid.UUID
	if existing != nil {
		itemID = *existing
		if _, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $2, sort_order = $3, parent_id = $4, published = TRUE, archived = FALSE, updated_at = NOW()
WHERE id = $1 AND course_id = $5 AND kind = 'content_page'
`, itemID, page.Title, page.SortOrder, moduleID, courseID); err != nil {
			return false, err
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived)
VALUES ($1, $2, 'content_page', $3, $4, TRUE, FALSE)
RETURNING id
`, courseID, page.SortOrder, page.Title, moduleID).Scan(&itemID)
		if err != nil {
			return false, err
		}
		needsBody = true
	}
	if needsBody {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.module_content_pages (structure_item_id, markdown, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (structure_item_id) DO UPDATE SET markdown = EXCLUDED.markdown, updated_at = NOW()
`, itemID, page.Markdown); err != nil {
			return false, err
		}
	}
	return needsBody, mcrepo.UpsertContentItem(ctx, tx, courseSlug, page.Slug, itemID, page.ContentVer, nil)
}

func upsertAssignment(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, courseSlug string, moduleID uuid.UUID, assign AssignmentFixture) (bool, error) {
	groupID, err := mcrepo.AssignmentGroupIDByName(ctx, tx, courseID, assign.Grading.Group)
	if err != nil {
		return false, err
	}
	allowText, allowFile, allowURL := submissionModeFlags(assign.Grading.SubmissionModes)

	existing, needsBody, err := itemNeedsSync(ctx, tx, courseSlug, assign.Slug, assign.ContentVer)
	if err != nil {
		return false, err
	}
	var itemID uuid.UUID
	if existing != nil {
		itemID = *existing
		if _, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $2, sort_order = $3, parent_id = $4, published = TRUE, archived = FALSE,
    assignment_group_id = $5, updated_at = NOW()
WHERE id = $1 AND course_id = $6 AND kind = 'assignment'
`, itemID, assign.Title, assign.SortOrder, moduleID, groupID, courseID); err != nil {
			return false, err
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived, assignment_group_id)
VALUES ($1, $2, 'assignment', $3, $4, TRUE, FALSE, $5)
RETURNING id
`, courseID, assign.SortOrder, assign.Title, moduleID, groupID).Scan(&itemID)
		if err != nil {
			return false, err
		}
		needsBody = true
	}
	if needsBody {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.module_assignments (
    structure_item_id, markdown, points_worth,
    submission_allow_text, submission_allow_file_upload, submission_allow_url,
    late_submission_policy, posting_policy, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, 'allow', 'automatic', NOW())
ON CONFLICT (structure_item_id) DO UPDATE SET
    markdown = EXCLUDED.markdown,
    points_worth = EXCLUDED.points_worth,
    submission_allow_text = EXCLUDED.submission_allow_text,
    submission_allow_file_upload = EXCLUDED.submission_allow_file_upload,
    submission_allow_url = EXCLUDED.submission_allow_url,
    late_submission_policy = EXCLUDED.late_submission_policy,
    posting_policy = EXCLUDED.posting_policy,
    updated_at = NOW()
`, itemID, assign.Markdown, assign.Grading.Points, allowText, allowFile, allowURL); err != nil {
			return false, err
		}
	}
	policy := assign.Grading.GradePolicy
	return needsBody, mcrepo.UpsertContentItem(ctx, tx, courseSlug, assign.Slug, itemID, assign.ContentVer, &policy)
}

func upsertQuiz(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, courseSlug string, moduleID uuid.UUID, quiz QuizFixture) (bool, error) {
	qJSON, err := json.Marshal(quiz.Questions)
	if err != nil {
		return false, err
	}
	groupID, err := mcrepo.AssignmentGroupIDByName(ctx, tx, courseID, quiz.Grading.Group)
	if err != nil {
		return false, err
	}

	existing, needsBody, err := itemNeedsSync(ctx, tx, courseSlug, quiz.Slug, quiz.ContentVer)
	if err != nil {
		return false, err
	}
	var itemID uuid.UUID
	if existing != nil {
		itemID = *existing
		if _, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $2, sort_order = $3, parent_id = $4, published = TRUE, archived = FALSE,
    assignment_group_id = $5, updated_at = NOW()
WHERE id = $1 AND course_id = $6 AND kind = 'quiz'
`, itemID, quiz.Title, quiz.SortOrder, moduleID, groupID, courseID); err != nil {
			return false, err
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived, assignment_group_id)
VALUES ($1, $2, 'quiz', $3, $4, TRUE, FALSE, $5)
RETURNING id
`, courseID, quiz.SortOrder, quiz.Title, moduleID, groupID).Scan(&itemID)
		if err != nil {
			return false, err
		}
		needsBody = true
	}
	if needsBody {
		maxAttempts := quiz.Grading.MaxAttempts
		unlimited := quiz.Grading.UnlimitedAttempts
		if unlimited {
			if maxAttempts <= 0 {
				maxAttempts = 1
			}
		} else if maxAttempts <= 0 {
			maxAttempts = 3
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.module_quizzes (
    structure_item_id, markdown, questions_json, points_worth,
    max_attempts, unlimited_attempts, grade_attempt_policy,
    show_score_timing, review_when, late_submission_policy, updated_at
) VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7, 'immediate', 'after_submit', 'allow', NOW())
ON CONFLICT (structure_item_id) DO UPDATE SET
    markdown = EXCLUDED.markdown,
    questions_json = EXCLUDED.questions_json,
    points_worth = EXCLUDED.points_worth,
    max_attempts = EXCLUDED.max_attempts,
    unlimited_attempts = EXCLUDED.unlimited_attempts,
    grade_attempt_policy = EXCLUDED.grade_attempt_policy,
    show_score_timing = EXCLUDED.show_score_timing,
    review_when = EXCLUDED.review_when,
    late_submission_policy = EXCLUDED.late_submission_policy,
    updated_at = NOW()
`, itemID, quiz.Markdown, qJSON, quiz.Grading.Points, maxAttempts, unlimited, quiz.Grading.GradeAttemptPolicy); err != nil {
			return false, err
		}
	}
	policy := quiz.Grading.GradePolicy
	return needsBody, mcrepo.UpsertContentItem(ctx, tx, courseSlug, quiz.Slug, itemID, quiz.ContentVer, &policy)
}

func submissionModeFlags(modes []string) (text, file, url bool) {
	text = true
	for _, m := range modes {
		switch strings.TrimSpace(strings.ToLower(m)) {
		case "text":
			text = true
		case "file", "file_upload", "upload":
			file = true
		case "url":
			url = true
		}
	}
	if !text && !file && !url {
		text = true
	}
	return text, file, url
}

func archiveRemovedItems(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, courseSlug string, desired map[string]struct{}) (int, error) {
	rows, err := mcrepo.ListContentItems(ctx, tx, courseSlug)
	if err != nil {
		return 0, err
	}
	var archived int
	for _, row := range rows {
		if _, keep := desired[row.Slug]; keep {
			continue
		}
		tag, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET archived = TRUE, published = FALSE, updated_at = NOW()
WHERE id = $1 AND course_id = $2 AND NOT archived
`, row.StructureItemID, courseID)
		if err != nil {
			return archived, err
		}
		archived += int(tag.RowsAffected())
	}
	return archived, nil
}
