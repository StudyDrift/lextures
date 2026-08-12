package marketingcontent

import (
	"testing"
	"time"
)

func TestPublicPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind, locale, cat, slug, want string
	}{
		{"blog", "en", "", "hello", "/blog/hello"},
		{"blog", "", "", "hello", "/blog/hello"},
		{"blog", "es", "", "hola", "/es/blog/hola"},
		{"doc", "en", "courses", "finding-your-course", "/docs/courses/finding-your-course"},
		{"doc", "es", "cursos", "encontrar-tu-curso", "/es/docs/cursos/encontrar-tu-curso"},
	}
	for _, tc := range cases {
		got := PublicPath(tc.kind, tc.locale, tc.cat, tc.slug)
		if got != tc.want {
			t.Fatalf("PublicPath(%q,%q,%q,%q)=%q want %q", tc.kind, tc.locale, tc.cat, tc.slug, got, tc.want)
		}
	}
	if got := PublicPath("blog", "../etc", "", "x"); got != "/blog/x" {
		t.Fatalf("unsafe locale leaked into path: %q", got)
	}
}

func TestNormalizeLocaleCodeRejectsTraversal(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"../etc", "en/../../x", "es\\win", ".", "..", "x", "this-is-way-too-long-to-be-a-tag"} {
		if got := NormalizeLocaleCode(raw); got != "" {
			t.Fatalf("NormalizeLocaleCode(%q)=%q, want empty", raw, got)
		}
	}
	if got := NormalizeLocaleCode("ES"); got != "es" {
		t.Fatalf("got %q", got)
	}
}

func TestTranslationIsStale(t *testing.T) {
	t.Parallel()
	src := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	synced := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	if !TranslationIsStale(&src, &synced) {
		t.Fatal("expected stale")
	}
	later := src.Add(time.Hour)
	if TranslationIsStale(&src, &later) {
		t.Fatal("did not expect stale")
	}
	if TranslationIsStale(nil, &synced) || TranslationIsStale(&src, nil) {
		t.Fatal("nil timestamps are not stale")
	}
}
