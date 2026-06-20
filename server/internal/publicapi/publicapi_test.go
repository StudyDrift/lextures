package publicapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	t.Parallel()
	for _, off := range []int{0, 25, 100} {
		c := EncodeCursor(off)
		got, err := DecodeCursor(c)
		if err != nil || got != off {
			t.Fatalf("offset %d: got %d err %v", off, got, err)
		}
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := DecodeCursor("not-valid"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSlicePage(t *testing.T) {
	t.Parallel()
	all := []int{1, 2, 3, 4, 5}
	slice, total := SlicePage(all, 1, 2)
	if total != 5 || len(slice) != 2 || slice[0] != 2 {
		t.Fatalf("got %v total %d", slice, total)
	}
}

func TestBuildCollectionResponse_LinkHeader(t *testing.T) {
	t.Parallel()
	items := []any{map[string]string{"id": "1"}, map[string]string{"id": "2"}}
	resp := BuildCollectionResponse(items, 10, 0, 2, "/api/v1/courses", nil)
	if resp.Meta.Total != 10 || resp.Links.Next == "" {
		t.Fatalf("meta/links: %+v", resp)
	}
	w := httptest.NewRecorder()
	SetLinkHeader(w, resp.Links)
	if w.Header().Get("Link") == "" {
		t.Fatal("expected Link header")
	}
}

func TestHasScope(t *testing.T) {
	t.Parallel()
	if !HasScope([]string{"courses:read", "grades:read"}, "courses:read") {
		t.Fatal("expected match")
	}
	if HasScope([]string{"courses:read"}, "grades:read") {
		t.Fatal("expected no match")
	}
}

func TestWriteUnauthorized_ProblemJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	WriteUnauthorized(w, "/api/v1/courses")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
	var p Problem
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil || p.Title != "Unauthorized" {
		t.Fatalf("body %+v err %v", p, err)
	}
}

func TestMatch_Routes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method, path string
		want         bool
	}{
		{http.MethodGet, "/api/v1/courses", true},
		{http.MethodGet, "/api/v1/openapi.json", true},
		{http.MethodGet, "/api/v1/courses/00000000-0000-0000-0000-000000000001", true},
		{http.MethodGet, "/api/v1/courses/C-ABC123", false},
		{http.MethodGet, "/api/v1/me", false},
	}
	for _, tc := range cases {
		_, _, ok := Match(tc.method, tc.path)
		if ok != tc.want {
			t.Fatalf("%s %s: got %v want %v", tc.method, tc.path, ok, tc.want)
		}
	}
}

func TestTokenLimiter_Deny(t *testing.T) {
	t.Parallel()
	l := NewTokenLimiter(1, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	if ok, _ := l.Allow("k", now); !ok {
		t.Fatal("first request should pass")
	}
	if ok, retry := l.Allow("k", now); ok || retry < 1 {
		t.Fatalf("second request should be denied, got ok=%v retry=%d", ok, retry)
	}
}

func TestApplyFieldset(t *testing.T) {
	t.Parallel()
	v := map[string]any{"id": "1", "title": "T", "secret": "x"}
	out := ApplyFieldset(v, map[string]struct{}{"title": {}}).(map[string]any)
	if out["title"] != "T" || out["id"] != "1" {
		t.Fatalf("got %v", out)
	}
	if _, ok := out["secret"]; ok {
		t.Fatal("secret should be stripped")
	}
}
