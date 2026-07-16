package board

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestValidateUnfurlURL_blocksPrivate(t *testing.T) {
	t.Parallel()
	for _, u := range []string{
		"http://127.0.0.1/",
		"http://localhost/x",
		"http://10.0.0.5/a",
		"http://192.168.1.1/",
		"http://[::1]/",
	} {
		err := ValidateUnfurlURL(u)
		if !errors.Is(err, ErrUnfurlSSRF) {
			t.Fatalf("%s: got %v, want ErrUnfurlSSRF", u, err)
		}
	}
}

func TestValidateUnfurlURL_allowsPublicHostShape(t *testing.T) {
	t.Parallel()
	if err := ValidateUnfurlURL("https://example.com/page"); err != nil {
		t.Fatalf("example.com: %v", err)
	}
}

func TestYouTubeAndVimeoIDs(t *testing.T) {
	t.Parallel()
	if id := YouTubeVideoID("https://www.youtube.com/watch?v=dQw4w9WgXcQ"); id != "dQw4w9WgXcQ" {
		t.Fatalf("yt watch = %q", id)
	}
	if id := YouTubeVideoID("https://youtu.be/dQw4w9WgXcQ"); id != "dQw4w9WgXcQ" {
		t.Fatalf("youtu.be = %q", id)
	}
	if id := VimeoVideoID("https://vimeo.com/123456789"); id != "123456789" {
		t.Fatalf("vimeo = %q", id)
	}
}

func TestParseOGMeta(t *testing.T) {
	t.Parallel()
	doc := `<!doctype html><html><head>
		<meta property="og:title" content="Hello Title"/>
		<meta property="og:description" content="Desc"/>
		<meta property="og:image" content="/img.png"/>
		<meta property="og:site_name" content="Site"/>
		<title>Fallback</title>
	</head><body></body></html>`
	n, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	p := parseOGMeta(n)
	if p.Title != "Hello Title" {
		t.Fatalf("title=%q", p.Title)
	}
	if p.Description != "Desc" {
		t.Fatalf("desc=%q", p.Description)
	}
	if p.SiteName != "Site" {
		t.Fatalf("site=%q", p.SiteName)
	}
}
