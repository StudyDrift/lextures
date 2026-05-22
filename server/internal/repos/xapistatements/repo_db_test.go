//go:build integration

package xapistatements_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/xapistatements"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestInsertAndList(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := uuid.New()
	stmtID := uuid.New()
	now := time.Now().UTC()
	title := "Quiz 1"
	row := xapistatements.Row{
		StatementID:     stmtID,
		ActorHash:       "abc",
		VerbID:          "http://adlnet.gov/expapi/verbs/passed",
		ObjectID:        "https://lextures.test/courses/demo/quiz/1",
		ObjectTitle:     &title,
		ContextCourseID: &courseID,
		StoredAt:        now,
		FullJSON:        json.RawMessage(`{"xapi":{},"caliper":{}}`),
	}
	if err := xapistatements.Insert(ctx, pool, row); err != nil {
		t.Fatal(err)
	}
	rows, err := xapistatements.ListForCourse(ctx, pool, courseID, now.Add(-time.Hour), now.Add(time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].StatementID != stmtID {
		t.Fatalf("got %+v", rows)
	}
}
