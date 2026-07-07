package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFilterLibraryBooks(t *testing.T) {
	author := "Ada"
	books := []libraryBook{
		{ID: "1", Title: "Algorithms"},
		{ID: "2", Title: "Poetry", Author: &author},
	}
	got := filterLibraryBooks(books, "ada")
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("filter = %+v", got)
	}
}

func TestCatalogPublish_Success(t *testing.T) {
	var saved catalogListing
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/api/v1/courses/CS101/catalog-listing" {
			_ = json.NewDecoder(r.Body).Decode(&saved)
			_ = json.NewEncoder(w).Encode(map[string]any{"listing": saved})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	catalogPublishFlags.course = "CS101"
	catalogPublishFlags.slug = "intro-cs"
	defer func() {
		catalogPublishFlags.course = ""
		catalogPublishFlags.slug = ""
	}()

	setCfg(srv.URL, "test-key")
	if err := catalogPublishCmd.RunE(catalogPublishCmd, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !saved.IsPublic || saved.Slug != "intro-cs" {
		t.Fatalf("saved = %+v", saved)
	}
}

func TestLibrarySearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/library/search" && r.URL.Query().Get("q") == "calculus" {
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{map[string]any{"title": "Calc"}}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	librarySearchCmd.SetOut(&out)
	if err := librarySearchCmd.RunE(librarySearchCmd, []string{"calculus"}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(out.String(), "Calc") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestOERSearch_RequiresProvider(t *testing.T) {
	err := oerSearchCmd.RunE(oerSearchCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--provider") {
		t.Fatalf("err = %v", err)
	}
}

func TestInclusiveAccessGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/inclusive-access" {
			_ = json.NewEncoder(w).Encode(map[string]any{"enabled": true, "isbn": "978-1"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	inclusiveAccessGetFlags.course = "CS101"
	defer func() { inclusiveAccessGetFlags.course = "" }()
	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	inclusiveAccessGetCmd.SetOut(&out)
	if err := inclusiveAccessGetCmd.RunE(inclusiveAccessGetCmd, nil); err != nil {
		t.Fatalf("inclusive access: %v", err)
	}
	if !strings.Contains(out.String(), "978-1") {
		t.Fatalf("output = %q", out.String())
	}
}
