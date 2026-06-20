package publicapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEncodeDecodeCursor(t *testing.T) {
	t.Parallel()
	c := EncodeCursor(25)
	off, err := DecodeCursor(c)
	if err != nil || off != 25 {
		t.Fatalf("round trip: off=%d err=%v", off, err)
	}
	if _, err := DecodeCursor("!!!"); err == nil {
		t.Fatal("expected invalid cursor error")
	}
}

func TestHasScope(t *testing.T) {
	t.Parallel()
	if !HasScope([]string{"courses:read", "grades:read"}, "courses:read") {
		t.Fatal("expected scope match")
	}
	if HasScope([]string{"courses:read"}, "grades:read") {
		t.Fatal("expected missing scope")
	}
}

func TestWriteProblem(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WriteProblem(rr, Problem{Type: "unauthorized", Title: "Unauthorized", Status: http.StatusUnauthorized, Detail: "nope", Instance: "/api/v1/courses"})
	if rr.Code != 401 {
		t.Fatalf("status: %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "problem+json") {
		t.Fatalf("content-type: %q", ct)
	}
	var p Problem
	if err := json.NewDecoder(rr.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Type, "https://lextures.io/errors/") {
		t.Fatalf("type: %q", p.Type)
	}
}

func TestAllowTokenRateLimit(t *testing.T) {
	t.Parallel()
	ResetRateLimits()
	if ok, _ := AllowToken("tok-a", 2); !ok {
		t.Fatal("first request should pass")
	}
	if ok, _ := AllowToken("tok-a", 2); !ok {
		t.Fatal("second request should pass")
	}
	if ok, retry := AllowToken("tok-a", 2); ok || retry < 1 {
		t.Fatalf("third should be limited, ok=%v retry=%d", ok, retry)
	}
}

func TestPaginateSlice(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5}
	page, next := PaginateSlice(items, 0, 2)
	if len(page) != 2 || next == "" {
		t.Fatalf("page=%v next=%q", page, next)
	}
	page, next = PaginateSlice(items, 4, 2)
	if len(page) != 1 || next != "" {
		t.Fatalf("tail page=%v next=%q", page, next)
	}
}

func TestFilterObjectSparseFields(t *testing.T) {
	t.Parallel()
	fields := map[string]struct{}{"title": {}}
	out := FilterObject(fields, map[string]any{"id": "1", "title": "A", "description": "x"})
	if _, ok := out["description"]; ok {
		t.Fatal("description should be omitted")
	}
	if out["id"] != "1" || out["title"] != "A" {
		t.Fatalf("out: %#v", out)
	}
}

func TestOpenAPI31ValidJSON(t *testing.T) {
	t.Parallel()
	var doc map[string]any
	if err := json.Unmarshal([]byte(OpenAPI31Document), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("version: %v", doc["openapi"])
	}
}
