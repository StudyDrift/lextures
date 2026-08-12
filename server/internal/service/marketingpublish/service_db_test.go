package marketingpublish

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecordChangeCoalescesAndUrgentBypassesDebounceDB(t *testing.T) {
	if testing.Short() {
		t.Skip("database integration test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var actor uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	article := uuid.New()
	path := "/blog/publish-test-" + article.String()
	author := "publish-test-" + article.String()
	if _, err = tx.Exec(ctx, `INSERT INTO marketing.content_authors(slug,name,created_by) VALUES($1,'Publish test',$2)`, author, actor); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO marketing.content_articles(id,kind,slug,path,title,status,author_slug,created_by) VALUES($1,'blog',$2,$3,'Publish test','draft',$4,$5)`, article, "publish-test-"+article.String(), path, author, actor); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM marketing.content_publish_events; DELETE FROM marketing.content_builds`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	first, err := RecordChange(ctx, tx, article, path, "publish", &actor, false, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RecordChange(ctx, tx, article, "/another-path", "update", &actor, false, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("builds did not coalesce: %s %s", first, second)
	}
	urgent, err := RecordChange(ctx, tx, article, path, "unpublish", &actor, true, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if urgent != first {
		t.Fatal("urgent change did not join pending build")
	}
	var count int
	var isUrgent bool
	var notBefore time.Time
	var paths []string
	if err = tx.QueryRow(ctx, `SELECT count(*),bool_or(urgent),min(not_before),array_agg(DISTINCT p ORDER BY p) FROM marketing.content_builds CROSS JOIN LATERAL unnest(paths) p`).Scan(&count, &isUrgent, &notBefore, &paths); err != nil {
		t.Fatal(err)
	}
	if count != 2 { // one build row appears once per distinct path in this aggregate
		t.Fatalf("coalesced path rows=%d paths=%v", count, paths)
	}
	if !isUrgent || notBefore.After(now.Add(2*time.Minute+time.Second)) {
		t.Fatalf("urgent=%v notBefore=%v", isUrgent, notBefore)
	}
	var events int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM marketing.content_publish_events WHERE build_id=$1`, first).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 3 {
		t.Fatalf("events=%d", events)
	}
}
