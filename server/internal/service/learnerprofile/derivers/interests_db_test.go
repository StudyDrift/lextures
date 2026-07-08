package derivers

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	learnerprofileservice "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func TestInterestsDeriver_InsufficientData_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openInterestsPool(t, ctx)
	defer pool.Close()

	userID := insertInterestsUser(t, ctx, pool)
	defer deleteInterestsUser(ctx, pool, userID)

	fix := seedInterestsCourse(t, ctx, pool, userID, "Ecology")
	insertNotebookGroup(t, ctx, pool, userID, globalStudentNotebookKey, "Ecology", "g1")

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := InterestsDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "insufficient_data" {
		t.Fatalf("state=%q want insufficient_data", result.State)
	}
	_ = fix
}

func TestInterestsDeriver_EcologySelfDirected_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openInterestsPool(t, ctx)
	defer pool.Close()

	userID := insertInterestsUser(t, ctx, pool)
	defer deleteInterestsUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	ecologyCourse := seedInterestsCourse(t, ctx, pool, userID, "Ecology")
	statsCourse := seedInterestsCourse(t, ctx, pool, userID, "Statistics")

	insertNotebookGroup(t, ctx, pool, userID, globalStudentNotebookKey, "Ecology", "g1")
	insertNotebookGroup(t, ctx, pool, userID, globalStudentNotebookKey, "Ecology", "g2")
	insertNotebookPage(t, ctx, pool, userID, globalStudentNotebookKey, "Ecology notes", "g1", "p1")
	insertNotebookGroup(t, ctx, pool, userID, globalStudentNotebookKey, "Ecology", "g3")

	pageID := uuid.New().String()
	insertNotebookTask(t, ctx, pool, userID, globalStudentNotebookKey, pageID, "Review field notes")
	insertNotebookPage(t, ctx, pool, userID, globalStudentNotebookKey, "More ecology", "g1", pageID)

	for i := 0; i < 4; i++ {
		insertInterestsFeedMessage(t, ctx, pool, userID, ecologyCourse.courseID, fixedNow.AddDate(0, 0, -i))
	}
	for i := 0; i < 2; i++ {
		insertInterestsFeedMessage(t, ctx, pool, userID, statsCourse.courseID, fixedNow.AddDate(0, 0, -i))
	}

	readingID := uuid.New()
	insertEngagementEvent(t, ctx, pool, userID, readingID, "content_page", "scroll_depth", 90, fixedNow.AddDate(0, 0, -1))
	insertEngagementEvent(t, ctx, pool, userID, readingID, "content_page", "time_on_task", 600, fixedNow.AddDate(0, 0, -1))
	if _, err := pool.Exec(ctx, `
UPDATE analytics.engagement_events SET course_id = $2
WHERE user_id = $1 AND item_id = $3
`, userID, ecologyCourse.courseID, readingID); err != nil {
		t.Fatal(err)
	}

	deriver := InterestsDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "ok" {
		t.Fatalf("state=%q", result.State)
	}

	var summary InterestsSummary
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Topics) == 0 || summary.Topics[0].Topic != "Ecology" {
		t.Fatalf("topics=%+v", summary.Topics)
	}
	if !summary.Topics[0].SelfDirected {
		t.Fatalf("expected ecology self-directed: %+v", summary.Topics[0])
	}
	if summary.Topics[0].Sources.Notebooks < 3 {
		t.Fatalf("notebook sources=%+v", summary.Topics[0].Sources)
	}

	svc := learnerprofileservice.New(pool, deriver)
	if err := svc.RecomputeIncremental(ctx, userID, "interests"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetFacet(ctx, userID, "interests")
	if err != nil || detail == nil {
		t.Fatalf("get facet: %v %+v", err, detail)
	}
	if detail.Facet.State != "ok" || len(detail.Insights) == 0 {
		t.Fatalf("stored facet=%+v insights=%d", detail.Facet, len(detail.Insights))
	}
}

func openInterestsPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	return pool
}

func insertInterestsUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := "interests-" + userID.String() + "@e.invalid"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name)
VALUES ($1, $2, 'hash', 'Interests Tester')
`, userID, email); err != nil {
		t.Fatal(err)
	}
	return userID
}

func deleteInterestsUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	_, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID)
}

type interestsCourseFixture struct {
	courseID uuid.UUID
}

func seedInterestsCourse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, category string) interestsCourseFixture {
	t.Helper()
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, catalog_category, created_by_user_id)
VALUES ($1, $2, $3, $4) RETURNING id
`, cc, category+" course", category, userID).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active)
VALUES ($1, $2, 'student', TRUE)
`, courseID, userID); err != nil {
		t.Fatal(err)
	}
	var channelID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.feed_channels (course_id, name, created_by_user_id)
VALUES ($1, 'General', $2) RETURNING id
`, courseID, userID).Scan(&channelID); err != nil {
		t.Fatal(err)
	}
	return interestsCourseFixture{courseID: courseID}
}

func insertNotebookGroup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode, title, groupID string) {
	t.Helper()
	upsertNotebookPages(t, ctx, pool, userID, courseCode, []notebookPage{
		{ID: groupID, Title: title, Kind: "group"},
	})
}

func insertNotebookPage(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode, content, parentID, pageID string) {
	t.Helper()
	parent := parentID
	upsertNotebookPages(t, ctx, pool, userID, courseCode, []notebookPage{
		{ID: pageID, ParentID: &parent, Kind: "page", ContentMd: content},
	})
}

func upsertNotebookPages(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode string, pages []notebookPage) {
	t.Helper()
	var existing notebookStore
	var raw []byte
	err := pool.QueryRow(ctx, `
SELECT data FROM analytics.student_notebooks WHERE user_id = $1 AND course_code = $2
`, userID, courseCode).Scan(&raw)
	if err == nil {
		_ = json.Unmarshal(raw, &existing)
	}
	byID := make(map[string]notebookPage, len(existing.Pages))
	for _, p := range existing.Pages {
		byID[p.ID] = p
	}
	for _, p := range pages {
		byID[p.ID] = p
	}
	merged := make([]notebookPage, 0, len(byID))
	for _, p := range byID {
		merged = append(merged, p)
	}
	store := notebookStore{Pages: merged}
	data, err := json.Marshal(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.student_notebooks (user_id, course_code, data, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (user_id, course_code) DO UPDATE SET data = EXCLUDED.data, updated_at = now()
`, userID, courseCode, data); err != nil {
		t.Fatal(err)
	}
}

func insertNotebookTask(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode, pageID, text string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.student_notebook_tasks (id, user_id, course_code, notebook_page_id, task_text)
VALUES ($1, $2, $3, $4, $5)
`, uuid.New(), userID, courseCode, pageID, text); err != nil {
		t.Fatal(err)
	}
}

func insertInterestsFeedMessage(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, at time.Time) {
	t.Helper()
	var channelID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT id FROM course.feed_channels WHERE course_id = $1 LIMIT 1
`, courseID).Scan(&channelID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.feed_messages (channel_id, author_user_id, body, created_at)
VALUES ($1, $2, 'Field observation thread', $3)
`, channelID, userID, at); err != nil {
		t.Fatal(err)
	}
}