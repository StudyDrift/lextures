package objectcache

import (
	"strings"
	"testing"

	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

func TestCourseStructureKey(t *testing.T) {
	if got := CourseStructureKey("abc", true); got != "cache:course:abc:structure:staff" {
		t.Fatalf("staff key: %q", got)
	}
	if got := CourseStructureKey("abc", false); got != "cache:course:abc:structure:student" {
		t.Fatalf("student key: %q", got)
	}
}

func TestCatalogPageKeyStable(t *testing.T) {
	f := repoCourse.PublicCatalogFilter{Q: "go", Limit: 20, Offset: 0}
	a := CatalogPageKey(f)
	b := CatalogPageKey(f)
	if a != b {
		t.Fatalf("expected stable key, got %q vs %q", a, b)
	}
	f2 := f
	f2.Q = "rust"
	if CatalogPageKey(f2) == a {
		t.Fatal("expected different key for different filter")
	}
}

func TestMarketplacePageKeyStable(t *testing.T) {
	f := repoCourse.MarketplaceFilter{Q: "go", Limit: 20, Offset: 0, FreeOnly: true}
	a := MarketplacePageKey(f)
	b := MarketplacePageKey(f)
	if a != b {
		t.Fatalf("expected stable key, got %q vs %q", a, b)
	}
	f2 := f
	f2.FreeOnly = false
	if MarketplacePageKey(f2) == a {
		t.Fatal("expected different key when free_only changes")
	}
	if strings.HasPrefix(a, "cache:catalog:") {
		t.Fatal("marketplace key must not share public catalog namespace")
	}
	if !strings.HasPrefix(a, "cache:marketplace:page:") {
		t.Fatalf("unexpected marketplace key prefix: %q", a)
	}
}

func TestUserCalendarKey(t *testing.T) {
	cid := "course-1"
	if got := UserCalendarKey("u1", &cid); got != "cache:user:u1:calendar:course:course-1" {
		t.Fatalf("course scoped: %q", got)
	}
	if got := UserCalendarKey("u1", nil); got != "cache:user:u1:calendar" {
		t.Fatalf("user scoped: %q", got)
	}
}
