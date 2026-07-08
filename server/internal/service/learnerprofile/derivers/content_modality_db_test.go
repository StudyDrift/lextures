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

func TestContentModalityDeriver_InsufficientData_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openContentModalityPool(t, ctx)
	defer pool.Close()

	userID := insertContentModalityUser(t, ctx, pool)
	defer deleteContentModalityUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := ContentModalityDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}

	videoID := uuid.New()
	insertEngagementEvent(t, ctx, pool, userID, videoID, "video", "video_progress", 90, fixedNow.AddDate(0, 0, -1))
	insertEngagementEvent(t, ctx, pool, userID, videoID, "video", "video_progress", 95, fixedNow.AddDate(0, 0, -2))

	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "insufficient_data" {
		t.Fatalf("state=%q want insufficient_data", result.State)
	}
}

func TestContentModalityDeriver_VideoPreferring_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openContentModalityPool(t, ctx)
	defer pool.Close()

	userID := insertContentModalityUser(t, ctx, pool)
	defer deleteContentModalityUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := ContentModalityDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}

	v1, v2, v3 := uuid.New(), uuid.New(), uuid.New()
	r1, r2 := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{v1, v2, v3} {
		insertEngagementEvent(t, ctx, pool, userID, id, "video", "video_progress", 90, fixedNow.AddDate(0, 0, -1))
	}
	insertEngagementEvent(t, ctx, pool, userID, r1, "content_page", "scroll_depth", 30, fixedNow.AddDate(0, 0, -2))
	insertEngagementEvent(t, ctx, pool, userID, r1, "content_page", "heartbeat", 0, fixedNow.AddDate(0, 0, -2))
	insertEngagementEvent(t, ctx, pool, userID, r2, "content_page", "scroll_depth", 35, fixedNow.AddDate(0, 0, -3))
	insertEngagementEvent(t, ctx, pool, userID, r2, "content_page", "heartbeat", 0, fixedNow.AddDate(0, 0, -3))

	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "ok" {
		t.Fatalf("state=%q", result.State)
	}
	var summary ModalitySummary
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.ModalityAffinity["video"] <= summary.ModalityAffinity["reading"] {
		t.Fatalf("affinity video=%v reading=%v", summary.ModalityAffinity["video"], summary.ModalityAffinity["reading"])
	}
	if summary.Pacing == "" {
		t.Fatal("expected pacing label")
	}

	svc := learnerprofileservice.New(pool, deriver)
	if err := svc.RecomputeIncremental(ctx, userID, "content_modality"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetFacet(ctx, userID, "content_modality")
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

func openContentModalityPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

func insertContentModalityUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := "content-modality-" + userID.String() + "@e.invalid"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name)
VALUES ($1, $2, 'hash', 'Modality Tester')
`, userID, email); err != nil {
		t.Fatal(err)
	}
	return userID
}

func deleteContentModalityUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	_, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID)
}

func insertEngagementEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, itemID uuid.UUID, itemType, eventType string, value float64, at time.Time) {
	t.Helper()
	v := float32(value)
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, item_id, item_type, event_type, value, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, userID, itemID, itemType, eventType, v, at); err != nil {
		t.Fatal(err)
	}
}