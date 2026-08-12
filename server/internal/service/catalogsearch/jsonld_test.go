package catalogsearch

import (
	"testing"

	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

func strp(s string) *string   { return &s }
func f64p(f float64) *float64 { return &f }

func TestBuildCourseJSONLD_RequiredFields(t *testing.T) {
	c := repoCourse.PublicCatalogCourse{
		Slug:            "intro-python",
		Title:           "Intro to Python",
		Description:     "Learn Python from scratch.",
		Language:        "en",
		PriceCents:      4999,
		InstructorName:  strp("Ada Lovelace"),
		Category:        strp("Programming"),
		AverageRating:   f64p(4.5),
		RatingCount:     5,
		EnrollmentCount: 1200,
	}
	ld := BuildCourseJSONLD(c, "https://lextures.com")

	if _, hasContext := ld["@context"]; hasContext {
		t.Fatalf("expected no @context (graph builder owns it), got %v", ld["@context"])
	}
	if ld["@type"] != "Course" {
		t.Fatalf("@type = %v", ld["@type"])
	}
	if ld["@id"] != "https://lextures.com/explore/intro-python#course" {
		t.Fatalf("@id = %v", ld["@id"])
	}
	if ld["name"] != "Intro to Python" {
		t.Fatalf("name = %v", ld["name"])
	}
	if ld["description"] != "Learn Python from scratch." {
		t.Fatalf("description = %v", ld["description"])
	}
	prov, ok := ld["provider"].(map[string]any)
	if !ok || prov["name"] != ProviderName {
		t.Fatalf("provider = %v", ld["provider"])
	}
	if ld["url"] != "https://lextures.com/explore/intro-python" {
		t.Fatalf("url = %v", ld["url"])
	}
	offer, ok := ld["offers"].(map[string]any)
	if !ok {
		t.Fatalf("missing offers: %v", ld["offers"])
	}
	if offer["price"] != "49.99" {
		t.Fatalf("price = %v", offer["price"])
	}
	if offer["priceCurrency"] != "USD" {
		t.Fatalf("priceCurrency = %v", offer["priceCurrency"])
	}
	if offer["category"] != "Paid" {
		t.Fatalf("offer category = %v", offer["category"])
	}
	inst, ok := ld["hasCourseInstance"].(map[string]any)
	if !ok {
		t.Fatalf("missing hasCourseInstance")
	}
	if inst["instructor"].(map[string]any)["name"] != "Ada Lovelace" {
		t.Fatalf("instructor = %v", inst["instructor"])
	}
	rating, ok := ld["aggregateRating"].(map[string]any)
	if !ok || rating["ratingValue"] != 4.5 {
		t.Fatalf("aggregateRating = %v", ld["aggregateRating"])
	}
}

func TestBuildCourseJSONLD_OmitsSparseRating(t *testing.T) {
	c := repoCourse.PublicCatalogCourse{Slug: "new", Title: "New", AverageRating: f64p(5), RatingCount: 4}
	ld := BuildCourseJSONLD(c, "https://lextures.com")
	if _, ok := ld["aggregateRating"]; ok {
		t.Fatal("aggregateRating must require at least five verified reviews")
	}
}

func TestBuildCourseJSONLD_FreeCourse(t *testing.T) {
	c := repoCourse.PublicCatalogCourse{Slug: "free-course", Title: "Free", PriceCents: 0}
	ld := BuildCourseJSONLD(c, "")
	offer := ld["offers"].(map[string]any)
	if offer["price"] != "0.00" {
		t.Fatalf("price = %v", offer["price"])
	}
	if offer["category"] != "Free" {
		t.Fatalf("category = %v", offer["category"])
	}
	if _, ok := ld["url"]; ok {
		t.Fatalf("expected no url when baseURL empty")
	}
	if _, ok := ld["aggregateRating"]; ok {
		t.Fatalf("expected no aggregateRating when unrated")
	}
}

func TestBuildCourseJSONLDAt_MarketplacePath(t *testing.T) {
	c := repoCourse.PublicCatalogCourse{Slug: "intro-python", Title: "Intro to Python"}
	ld := BuildCourseJSONLDAt(c, "https://lextures.com/", "/courses/")
	if ld["url"] != "https://lextures.com/courses/intro-python" {
		t.Fatalf("url = %v", ld["url"])
	}
}
