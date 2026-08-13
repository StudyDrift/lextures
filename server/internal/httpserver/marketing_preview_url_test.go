package httpserver

import "testing"

func TestMarketingPreviewURL(t *testing.T) {
	t.Parallel()
	got := marketingPreviewURL("https://lextures.com", "/docs/courses/building-modules-and-pages", "tok+1")
	want := "https://lextures.com/docs/courses/building-modules-and-pages?preview_token=tok%2B1"
	if got != want {
		t.Fatalf("marketingPreviewURL() = %q, want %q", got, want)
	}
}

func TestMarketingPreviewURLDefaultsOriginAndSlash(t *testing.T) {
	t.Parallel()
	got := marketingPreviewURL("", "blog/hello", "abc")
	want := "https://lextures.com/blog/hello?preview_token=abc"
	if got != want {
		t.Fatalf("marketingPreviewURL() = %q, want %q", got, want)
	}
}
