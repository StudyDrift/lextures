package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

func TestPublicMarketplace_FeatureOff_Returns404(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: false}, JWTSigner: nil})
	for _, path := range []string{
		"/api/v1/public/marketplace/courses",
		"/api/v1/public/marketplace/categories",
		"/api/v1/public/marketplace/courses/some-slug",
		"/api/v1/public/marketplace/courses/some-slug/reviews",
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s: got %d, want 404 (body %s)", path, rr.Code, rr.Body.String())
		}
		var body struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: decode: %v", path, err)
		}
		if body.Error.Code != "NOT_FOUND" {
			t.Fatalf("%s: code = %q want NOT_FOUND (body %s)", path, body.Error.Code, rr.Body.String())
		}
	}
}

func TestPublicMarketplace_NoAuthRequired_WhenEnabled(t *testing.T) {
	// With flag on and no pool, list hits DB and returns 500 — but NOT 401.
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: true}, JWTSigner: nil})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/marketplace/courses", nil)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("public marketplace must not require auth; got 401")
	}
	if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusOK {
		// nil pool → 500 is expected; anything else unexpected.
		t.Fatalf("got %d, want 500 or 200 (body %s)", rr.Code, rr.Body.String())
	}
}

func TestPublicMarketplace_InvalidParams_Returns400(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: true}, JWTSigner: nil})
	cases := []string{
		"/api/v1/public/marketplace/courses?sort=bogus",
		"/api/v1/public/marketplace/courses?level=expert",
		"/api/v1/public/marketplace/courses?price_max=-1",
		"/api/v1/public/marketplace/courses?cursor=!!!",
		"/api/v1/public/marketplace/courses?free_only=maybe",
	}
	for _, path := range cases {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d, want 400 (body %s)", path, rr.Code, rr.Body.String())
		}
	}
}

func TestPublicMarketplace_CORSHeaderPresent(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: false}, JWTSigner: nil})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/marketplace/courses", nil)
	req.Header.Set("Origin", "https://lextures.com")
	h.ServeHTTP(rr, req)
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestToPublicMarketplaceCourse_OmitsOwned(t *testing.T) {
	c := repoCourse.MarketplaceCourse{
		ID:         "11111111-1111-1111-1111-111111111111",
		Slug:       "intro",
		CourseCode: "INTRO",
		Title:      "Intro",
		Owned:      true,
		PriceCents: 0,
	}
	pub := toPublicMarketplaceCourse(c)
	raw, err := json.Marshal(pub)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"owned"`) {
		t.Fatalf("public course JSON must omit owned: %s", raw)
	}
	if pub.Slug != "intro" || pub.Title != "Intro" {
		t.Fatalf("unexpected public course: %+v", pub)
	}
}

func TestMarketplaceCourseToPublicCatalog_MapsLevel(t *testing.T) {
	level := "beginner"
	c := repoCourse.MarketplaceCourse{Slug: "s", Title: "T", Level: &level, PriceCents: 100}
	pc := marketplaceCourseToPublicCatalog(c)
	if pc.DifficultyLevel == nil || *pc.DifficultyLevel != "beginner" {
		t.Fatalf("DifficultyLevel = %v", pc.DifficultyLevel)
	}
	if pc.PriceCents != 100 {
		t.Fatalf("PriceCents = %d", pc.PriceCents)
	}
}
