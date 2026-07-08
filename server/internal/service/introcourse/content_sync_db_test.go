package introcourse

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

func TestSyncContent_CreatesSevenModules_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{
		IntroCourseEnabled:           true,
		LearnerProfileEnabled:        true,
		PushNotificationsEnabled:     true,
		AdaptiveLearnerModelEnabled:  true,
		SRSPracticeEnabled:           true,
		DiagnosticAssessmentsEnabled: true,
		SelfReflectionEnabled:        true,
		AiDisclosureEnabled:          true,
	}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	var moduleCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_structure_items
WHERE course_id = $1 AND kind = 'module' AND parent_id IS NULL AND NOT archived
`, course.ID).Scan(&moduleCount); err != nil {
		t.Fatal(err)
	}
	if moduleCount != 7 {
		t.Fatalf("expected 7 published modules, got %d", moduleCount)
	}

	var pageCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_structure_items csi
INNER JOIN course.module_content_pages mcp ON mcp.structure_item_id = csi.id
WHERE csi.course_id = $1 AND csi.kind = 'content_page' AND NOT csi.archived
`, course.ID).Scan(&pageCount); err != nil {
		t.Fatal(err)
	}
	if pageCount < 20 {
		t.Fatalf("expected at least 20 content pages, got %d", pageCount)
	}

	var quizCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_structure_items
WHERE course_id = $1 AND kind = 'quiz' AND NOT archived
`, course.ID).Scan(&quizCount); err != nil {
		t.Fatal(err)
	}
	if quizCount != 7 {
		t.Fatalf("expected 7 quizzes, got %d", quizCount)
	}

	var sectionCount int
	if err := pool.QueryRow(ctx, `
SELECT jsonb_array_length(COALESCE(cs.sections, '[]'::jsonb))
FROM course.course_syllabus cs
WHERE cs.course_id = $1
`, course.ID).Scan(&sectionCount); err != nil {
		t.Fatal(err)
	}
	if sectionCount != 5 {
		t.Fatalf("expected 5 syllabus sections, got %d", sectionCount)
	}
}

func TestSyncContent_IdempotentNoop_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.SyncContentForCourse(ctx, cfg, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.SyncContentForCourse(ctx, cfg, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Skipped {
		t.Fatalf("expected second sync noop, got %+v (first %+v)", second, first)
	}
}

func TestSyncContent_LearnerProfileFlagArchivesModule_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)

	onCfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}
	course, err := svc.EnsureProvisioned(ctx, onCfg)
	if err != nil {
		t.Fatal(err)
	}

	offCfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: false}
	report, err := svc.SyncContentForCourse(ctx, offCfg, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.Archived == 0 {
		t.Fatal("expected archived items when learner profile disabled")
	}

	var archivedModule int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM settings.intro_course_items ici
INNER JOIN course.course_structure_items csi ON csi.id = ici.structure_item_id
WHERE ici.slug = 'm4.learner-profile' AND csi.archived
`).Scan(&archivedModule); err != nil {
		t.Fatal(err)
	}
	if archivedModule != 1 {
		t.Fatalf("expected learner profile module archived, got count=%d", archivedModule)
	}

	var otherGradeCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_grades cg
INNER JOIN course.course_structure_items csi ON csi.id = cg.module_item_id
WHERE csi.course_id = $1
`, course.ID).Scan(&otherGradeCount); err != nil && err.Error() != "" {
		_ = otherGradeCount
	}
	_ = uuid.Nil
}