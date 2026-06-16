package course

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

// TestPublicCatalog_Isolation_Pg verifies that only published, public, non-archived
// courses surface in the public catalog (AC-1, AC-5, FR security NFR).
func TestPublicCatalog_Isolation_Pg(t *testing.T) {
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

	em := "pcat-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("password1230")
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(u.ID)

	// A published, public course (visible) and a draft course (hidden).
	pub, err := CreateCourse(ctx, pool, uid, "Quantum Computing Basics", "Learn qubits.", "traditional", nil, nil, nil)
	if err != nil {
		t.Fatalf("create public: %v", err)
	}
	draft, err := CreateCourse(ctx, pool, uid, "Secret Draft Course", "Hidden.", "traditional", nil, nil, nil)
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE course.courses
		SET is_public = TRUE, published = TRUE, catalog_category = 'Science',
		    difficulty_level = 'beginner', price_cents = 0, enrollment_count = 5,
		    catalog_slug = 'quantum-computing-basics'
		WHERE course_code = $1`, pub.CourseCode); err != nil {
		t.Fatalf("publish public: %v", err)
	}
	// Draft stays is_public = FALSE, published = FALSE.

	list, total, _, err := ListPublicCatalog(ctx, pool, PublicCatalogFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, c := range list {
		if c.CourseCode == draft.CourseCode {
			t.Fatalf("draft course leaked into public catalog")
		}
	}
	found := false
	for _, c := range list {
		if c.CourseCode == pub.CourseCode {
			found = true
			if c.Slug != "quantum-computing-basics" {
				t.Fatalf("slug = %q", c.Slug)
			}
			if c.Category == nil || *c.Category != "Science" {
				t.Fatalf("category = %v", c.Category)
			}
		}
	}
	if !found {
		t.Fatalf("published public course missing from catalog (total=%d)", total)
	}

	// Search by title should find the public course.
	res, _, _, err := ListPublicCatalog(ctx, pool, PublicCatalogFilter{Q: "Quantum", Sort: CatalogSortRelevance})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("search for Quantum returned nothing")
	}

	// Detail by slug works for public, and returns nil for the draft's code.
	got, err := GetPublicCourseBySlug(ctx, pool, "quantum-computing-basics")
	if err != nil || got == nil {
		t.Fatalf("detail by slug: got %v err %v", got, err)
	}
	draftGot, err := GetPublicCourseBySlug(ctx, pool, draft.CourseCode)
	if err != nil {
		t.Fatalf("detail draft: %v", err)
	}
	if draftGot != nil {
		t.Fatalf("draft course reachable via public detail endpoint")
	}

	// Free filter: price_max = 0 keeps the free public course.
	zero := 0
	freeList, _, _, err := ListPublicCatalog(ctx, pool, PublicCatalogFilter{PriceMax: &zero})
	if err != nil {
		t.Fatalf("free filter: %v", err)
	}
	if len(freeList) == 0 {
		t.Fatalf("free filter excluded the $0 course")
	}

	// Categories taxonomy includes Science.
	cats, err := ListCatalogCategories(ctx, pool)
	if err != nil {
		t.Fatalf("categories: %v", err)
	}
	hasSci := false
	for _, c := range cats {
		if c.Category == "Science" {
			hasSci = true
		}
	}
	if !hasSci {
		t.Fatalf("Science category missing: %v", cats)
	}
}
