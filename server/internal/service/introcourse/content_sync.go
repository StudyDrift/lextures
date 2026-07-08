package introcourse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
)

// ContentSyncReport summarizes a content sync run.
type ContentSyncReport struct {
	Skipped        bool
	Modules        int
	Pages          int
	Assignments    int
	Quizzes        int
	Archived       int
	ContentVersion int
}

// SyncContent upserts curriculum fixtures into the intro course (idempotent, soft-archives removals).
func SyncContent(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, cfg config.Config) (ContentSyncReport, error) {
	started := time.Now()
	report := ContentSyncReport{ContentVersion: ContentVersion}

	cur, err := LoadCurriculum(defaultLocale)
	if err != nil {
		recordContentSync("error", started)
		return report, err
	}
	if err := ValidateCurriculum(cur); err != nil {
		recordContentSync("error", started)
		return report, err
	}

	storedVer, err := icrepo.MaxStoredContentVersion(ctx, tx)
	if err != nil {
		recordContentSync("error", started)
		return report, err
	}
	modules := FilterCurriculum(cur, cfg)
	desired := make(map[string]struct{})
	for _, mod := range modules {
		desired[mod.Meta.Slug] = struct{}{}
		for _, p := range mod.Pages {
			desired[p.Slug] = struct{}{}
		}
		for _, a := range mod.Assignments {
			desired[a.Slug] = struct{}{}
		}
		for _, q := range mod.Quizzes {
			desired[q.Slug] = struct{}{}
		}
	}

	if storedVer == ContentVersion {
		upToDate, uerr := curriculumUpToDate(ctx, tx, courseID, desired)
		if uerr != nil {
			recordContentSync("error", started)
			return report, uerr
		}
		syllabusOK, serr := syllabusUpToDate(ctx, tx, courseID)
		if serr != nil {
			recordContentSync("error", started)
			return report, serr
		}
		if upToDate && syllabusOK {
			report.Skipped = true
			recordContentSync("noop", started)
			setContentVersionGauge(ContentVersion)
			return report, nil
		}
	}

	for _, mod := range modules {
		moduleID, err := upsertModule(ctx, tx, courseID, mod.Meta)
		if err != nil {
			recordContentSync("error", started)
			return report, fmt.Errorf("module %s: %w", mod.Meta.Slug, err)
		}
		report.Modules++

		// Quizzes and pages before new assignments so sort_order moves free slots first.
		for _, quiz := range mod.Quizzes {
			if err := upsertQuiz(ctx, tx, courseID, moduleID, quiz); err != nil {
				recordContentSync("error", started)
				return report, fmt.Errorf("quiz %s: %w", quiz.Slug, err)
			}
			report.Quizzes++
		}
		for _, page := range mod.Pages {
			if err := upsertContentPage(ctx, tx, courseID, moduleID, page); err != nil {
				recordContentSync("error", started)
				return report, fmt.Errorf("page %s: %w", page.Slug, err)
			}
			report.Pages++
		}
		for _, assign := range mod.Assignments {
			if err := upsertAssignment(ctx, tx, courseID, moduleID, assign); err != nil {
				recordContentSync("error", started)
				return report, fmt.Errorf("assignment %s: %w", assign.Slug, err)
			}
			report.Assignments++
		}
	}

	archived, err := archiveRemovedItems(ctx, tx, courseID, desired)
	if err != nil {
		recordContentSync("error", started)
		return report, err
	}
	report.Archived = archived

	if err := SyncGradingConfig(ctx, tx, courseID, cfg); err != nil {
		recordContentSync("error", started)
		return report, fmt.Errorf("grading config: %w", err)
	}

	if err := syncSyllabus(ctx, tx, courseID); err != nil {
		recordContentSync("error", started)
		return report, fmt.Errorf("syllabus: %w", err)
	}

	recordContentSync("success", started)
	setContentVersionGauge(ContentVersion)
	return report, nil
}

func curriculumUpToDate(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, desired map[string]struct{}) (bool, error) {
	for slug := range desired {
		itemID, err := icrepo.LookupContentItem(ctx, tx, slug)
		if err != nil {
			return false, err
		}
		if itemID == nil {
			return false, nil
		}
		var version int
		var archived bool
		err = tx.QueryRow(ctx, `
SELECT ici.content_version, csi.archived
FROM settings.intro_course_items ici
INNER JOIN course.course_structure_items csi ON csi.id = ici.structure_item_id
WHERE ici.slug = $1 AND csi.course_id = $2
`, slug, courseID).Scan(&version, &archived)
		if err != nil {
			return false, err
		}
		if archived || version != ContentVersion {
			return false, nil
		}
	}

	rows, err := icrepo.ListContentItems(ctx, tx)
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

func upsertModule(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, meta ModuleMeta) (uuid.UUID, error) {
	existing, err := icrepo.LookupContentItem(ctx, tx, meta.Slug)
	if err != nil {
		return uuid.Nil, err
	}
	if existing != nil {
		if _, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $2, sort_order = $3, published = TRUE, archived = FALSE, updated_at = NOW()
WHERE id = $1 AND course_id = $4 AND kind = 'module' AND parent_id IS NULL
`, *existing, meta.Title, meta.SortOrder, courseID); err != nil {
			return uuid.Nil, err
		}
		if err := icrepo.UpsertContentItem(ctx, tx, meta.Slug, *existing, ContentVersion, nil); err != nil {
			return uuid.Nil, err
		}
		return *existing, nil
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived)
VALUES ($1, $2, 'module', $3, NULL, TRUE, FALSE)
RETURNING id
`, courseID, meta.SortOrder, meta.Title).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, icrepo.UpsertContentItem(ctx, tx, meta.Slug, id, ContentVersion, nil)
}

func upsertContentPage(ctx context.Context, tx pgx.Tx, courseID, moduleID uuid.UUID, page PageFixture) error {
	existing, err := icrepo.LookupContentItem(ctx, tx, page.Slug)
	if err != nil {
		return err
	}
	var itemID uuid.UUID
	if existing != nil {
		itemID = *existing
		if _, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $2, sort_order = $3, parent_id = $4, published = TRUE, archived = FALSE, updated_at = NOW()
WHERE id = $1 AND course_id = $5 AND kind = 'content_page'
`, itemID, page.Title, page.SortOrder, moduleID, courseID); err != nil {
			return err
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived)
VALUES ($1, $2, 'content_page', $3, $4, TRUE, FALSE)
RETURNING id
`, courseID, page.SortOrder, page.Title, moduleID).Scan(&itemID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.module_content_pages (structure_item_id, markdown)
VALUES ($1, $2)
ON CONFLICT (structure_item_id) DO NOTHING
`, itemID, page.Markdown); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO course.module_content_pages (structure_item_id, markdown, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (structure_item_id) DO UPDATE SET markdown = EXCLUDED.markdown, updated_at = NOW()
`, itemID, page.Markdown); err != nil {
		return err
	}
	return icrepo.UpsertContentItem(ctx, tx, page.Slug, itemID, ContentVersion, nil)
}

func upsertAssignment(ctx context.Context, tx pgx.Tx, courseID, moduleID uuid.UUID, assign AssignmentFixture) error {
	groupID, err := icrepo.AssignmentGroupIDByName(ctx, tx, courseID, assign.Grading.Group)
	if err != nil {
		return err
	}
	allowText, allowFile, allowURL := submissionModeFlags(assign.Grading.SubmissionModes)

	existing, err := icrepo.LookupContentItem(ctx, tx, assign.Slug)
	if err != nil {
		return err
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
			return err
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived, assignment_group_id)
VALUES ($1, $2, 'assignment', $3, $4, TRUE, FALSE, $5)
RETURNING id
`, courseID, assign.SortOrder, assign.Title, moduleID, groupID).Scan(&itemID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.module_assignments (structure_item_id, markdown, points_worth)
VALUES ($1, $2, $3)
ON CONFLICT (structure_item_id) DO NOTHING
`, itemID, assign.Markdown, assign.Grading.Points); err != nil {
			return err
		}
	}
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
		return err
	}
	policy := assign.Grading.GradePolicy
	return icrepo.UpsertContentItem(ctx, tx, assign.Slug, itemID, ContentVersion, &policy)
}

func upsertQuiz(ctx context.Context, tx pgx.Tx, courseID, moduleID uuid.UUID, quiz QuizFixture) error {
	qJSON, err := json.Marshal(quiz.Questions)
	if err != nil {
		return err
	}
	groupID, err := icrepo.AssignmentGroupIDByName(ctx, tx, courseID, quiz.Grading.Group)
	if err != nil {
		return err
	}

	existing, err := icrepo.LookupContentItem(ctx, tx, quiz.Slug)
	if err != nil {
		return err
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
			return err
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived, assignment_group_id)
VALUES ($1, $2, 'quiz', $3, $4, TRUE, FALSE, $5)
RETURNING id
`, courseID, quiz.SortOrder, quiz.Title, moduleID, groupID).Scan(&itemID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json)
VALUES ($1, $2, $3::jsonb)
ON CONFLICT (structure_item_id) DO NOTHING
`, itemID, quiz.Markdown, qJSON); err != nil {
			return err
		}
	}
	maxAttempts := quiz.Grading.MaxAttempts
	unlimited := false
	if maxAttempts <= 0 {
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
		return err
	}
	policy := quiz.Grading.GradePolicy
	return icrepo.UpsertContentItem(ctx, tx, quiz.Slug, itemID, ContentVersion, &policy)
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

func archiveRemovedItems(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, desired map[string]struct{}) (int, error) {
	rows, err := icrepo.ListContentItems(ctx, tx)
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