package managedlearners

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestSearch_ManagedLearnerWithoutEnrollment_Pg(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	stamp := time.Now().Format("20060102150405.000")
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	parentRow, err := user.InsertUser(ctx, pool, "ml-search-"+stamp+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("parent: %v", err)
	}
	parentID, _ := uuid.Parse(parentRow.ID)
	otherRow, err := user.InsertUser(ctx, pool, "ml-search-other-"+stamp+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	otherID, _ := uuid.Parse(otherRow.ID)

	grade := "4"
	dep, err := Create(ctx, pool, CreateParams{
		ActorID:     parentID,
		DisplayName: "QA Hunt Kid",
		GradeLevel:  &grade,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	hits, total, err := Search(ctx, pool, parentID, "QA Hunt Kid", nil, 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(hits) != 1 {
		t.Fatalf("hits=%d total=%d", len(hits), total)
	}
	if hits[0].UserID.String() != dep.ID || hits[0].DisplayName != "QA Hunt Kid" {
		t.Fatalf("hit: %+v", hits[0])
	}
	if hits[0].GradeLevel == nil || *hits[0].GradeLevel != "4" {
		t.Fatalf("grade: %+v", hits[0].GradeLevel)
	}
	if hits[0].Score < 0.9 {
		t.Fatalf("score: %v", hits[0].Score)
	}

	partial, partialTotal, err := Search(ctx, pool, parentID, "Hunt Kid", nil, 5)
	if err != nil {
		t.Fatalf("partial: %v", err)
	}
	if partialTotal != 1 || len(partial) != 1 || partial[0].UserID.String() != dep.ID {
		t.Fatalf("partial hits=%d total=%d", len(partial), partialTotal)
	}

	scope := "NOT-A-COURSE"
	scoped, scopedTotal, err := Search(ctx, pool, parentID, "QA Hunt Kid", &scope, 5)
	if err != nil {
		t.Fatalf("scope: %v", err)
	}
	if scopedTotal != 0 || len(scoped) != 0 {
		t.Fatalf("scoped hits=%d total=%d", len(scoped), scopedTotal)
	}

	hidden, hiddenTotal, err := Search(ctx, pool, otherID, "QA Hunt Kid", nil, 5)
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	if hiddenTotal != 0 || len(hidden) != 0 {
		t.Fatalf("other parent saw %d hits", len(hidden))
	}
}
