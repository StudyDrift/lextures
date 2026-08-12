package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

func TestPublicContentHashIgnoresUpdatedAtButTracksBody(t *testing.T) {
	now := time.Now().UTC()
	a := mcrepo.PublicArticle{Article: mcrepo.Article{
		ID: uuid.New(), Kind: "blog", Slug: "hello", Locale: "en", Path: "/blog/hello",
		Title: "Hello", Description: "Description", BodyMD: "first", TranslationGroupID: uuid.New(),
		AuthorSlug: "writer", PublishedAt: &now, UpdatedAt: now,
	}, Tags: []string{"news"}}
	first := mapPublicArticle(a, false, true).ContentHash
	a.UpdatedAt = now.Add(time.Hour)
	if got := mapPublicArticle(a, false, true).ContentHash; got != first {
		t.Fatalf("updatedAt-only touch changed hash: %s != %s", got, first)
	}
	a.BodyMD = "second"
	if got := mapPublicArticle(a, false, true).ContentHash; got == first {
		t.Fatal("body change did not change hash")
	}
}

func TestPublicContentStrongETagReturns304(t *testing.T) {
	payload := map[string]string{"title": "stable"}
	first := httptest.NewRecorder()
	writeJSONWithETag(first, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK, payload)
	tag := first.Header().Get("ETag")
	if tag == "" || len(tag) < 2 || tag[:2] == `W/` {
		t.Fatalf("expected strong ETag, got %q", tag)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", tag)
	second := httptest.NewRecorder()
	writeJSONWithETag(second, req, http.StatusOK, payload)
	if second.Code != http.StatusNotModified || second.Body.Len() != 0 {
		t.Fatalf("got status=%d body=%q", second.Code, second.Body.String())
	}
}
