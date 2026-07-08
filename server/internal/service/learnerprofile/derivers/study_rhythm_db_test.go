package derivers

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	learnerprofileservice "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func TestStudyRhythmDeriver_InsufficientData_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openStudyRhythmPool(t, ctx)
	defer pool.Close()

	userID := insertStudyRhythmUser(t, ctx, pool, "UTC")
	defer deleteStudyRhythmUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := StudyRhythmDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	for _, offset := range []int{1, 3} {
		insertHeartbeat(t, ctx, pool, userID, fixedNow.AddDate(0, 0, -offset))
	}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "insufficient_data" {
		t.Fatalf("state=%q want insufficient_data", result.State)
	}
}

func TestStudyRhythmDeriver_FullRhythm_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openStudyRhythmPool(t, ctx)
	defer pool.Close()

	userID := insertStudyRhythmUser(t, ctx, pool, "America/Denver")
	defer deleteStudyRhythmUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := StudyRhythmDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	loc, _ := time.LoadLocation("America/Denver")

	for day := 0; day < 12; day++ {
		localDay := time.Date(2026, 4, 1, 0, 0, 0, 0, loc).AddDate(0, 0, day)
		for i := 0; i < 70; i++ {
			localAt := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 19, 0, 0, 0, loc).
				Add(time.Duration(i) * 30 * time.Second)
			insertHeartbeat(t, ctx, pool, userID, localAt.UTC())
		}
	}

	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "ok" {
		t.Fatalf("state=%q", result.State)
	}
	var summary RhythmSummary
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.PeakWindows) == 0 || summary.PeakWindows[0].Dow != "weekday" {
		t.Fatalf("peak windows: %+v", summary.PeakWindows)
	}
	if summary.PeakWindows[0].HourBucket != "18-21" {
		t.Fatalf("hourBucket=%q want 18-21 (7pm local)", summary.PeakWindows[0].HourBucket)
	}
	if summary.LongestStreakDays < 12 {
		t.Fatalf("longestStreak=%d want >= 12", summary.LongestStreakDays)
	}
	if summary.CurrentStreakDays != 0 {
		t.Fatalf("currentStreak=%d want 0 after gap", summary.CurrentStreakDays)
	}
	if summary.MedianSessionMin < 34 || summary.MedianSessionMin > 36 {
		t.Fatalf("medianSessionMin=%d want ~35", summary.MedianSessionMin)
	}

	svc := learnerprofileservice.New(pool, deriver)
	if err := svc.RecomputeIncremental(ctx, userID, "study_rhythm"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetFacet(ctx, userID, "study_rhythm")
	if err != nil || detail == nil {
		t.Fatalf("get facet: %v %+v", err, detail)
	}
	if detail.Facet.State != "ok" {
		t.Fatalf("stored facet state=%q", detail.Facet.State)
	}
	if len(detail.Insights) == 0 {
		t.Fatal("expected stored insights")
	}
}

func openStudyRhythmPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

func insertStudyRhythmUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tz string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := "study-rhythm-" + userID.String() + "@e.invalid"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name, timezone)
VALUES ($1, $2, 'hash', 'Rhythm Tester', $3)
`, userID, email, tz); err != nil {
		t.Fatal(err)
	}
	return userID
}

func deleteStudyRhythmUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	_, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID)
}

func insertHeartbeat(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, at time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, event_type, occurred_at)
VALUES ($1, 'heartbeat', $2)
`, userID, at); err != nil {
		t.Fatal(err)
	}
}