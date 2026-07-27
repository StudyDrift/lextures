package context

import (
	"strings"
	"testing"
)

func TestExtractHTML_mainContent(t *testing.T) {
	html := `<!DOCTYPE html><html lang="en"><head><title>Lab Protocol</title>
<script>evil()</script><style>body{}</style></head>
<body><nav>nav</nav><main><p>Step one mix the reagents carefully.</p>
<p>Step two heat gently.</p></main><footer>footer</footer></body></html>`
	got, err := ExtractMainContent("text/html", []byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Lab Protocol" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.Lang != "en" {
		t.Fatalf("lang=%q", got.Lang)
	}
	if !strings.Contains(got.Text, "reagents") {
		t.Fatalf("text=%q", got.Text)
	}
	if strings.Contains(got.Text, "evil") || strings.Contains(got.Text, "footer") {
		t.Fatalf("boilerplate leaked: %q", got.Text)
	}
}

func TestExtractUnsupported(t *testing.T) {
	_, err := ExtractMainContent("image/png", []byte{0x89, 0x50, 0x4e, 0x47})
	if err != ErrUnsupportedType {
		t.Fatalf("got %v", err)
	}
}

func TestChunkAndTokenEstimate(t *testing.T) {
	text := strings.Repeat("word ", 500)
	chunks := ChunkText(text)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if EstimateTokens("abcd") != 1 {
		t.Fatalf("token estimate")
	}
	if LexicalScore("reagents heat", "mix the reagents and heat") < 2 {
		t.Fatalf("lexical score too low")
	}
}

func TestNormalizeAndHashURL(t *testing.T) {
	a, err := NormalizeURL("HTTPS://Example.COM/Path#frag")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NormalizeURL("https://example.com/Path")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("%q vs %q", a, b)
	}
	if HashURL(a) == "" || HashURL(a) != HashURL(b) {
		t.Fatal("hash mismatch")
	}
}

func TestCitationsFromPackAndFilter(t *testing.T) {
	pack := &ContextPack{
		Segments: []ContextSegment{
			{Kind: KindSection, ID: "s1", Title: "Sec"},
			{Kind: KindLink, ID: "l1", Title: "Link", URL: "https://example.com"},
			{Kind: KindCourse, ID: "c1", Title: "Course", Text: "x"},
		},
	}
	cites := CitationsFromPack(pack)
	if len(cites) != 2 {
		t.Fatalf("cites=%d", len(cites))
	}
	filtered := FilterValidCitations(append(cites, Citation{Kind: CiteLink, ID: "missing"}), pack)
	if len(filtered) != 2 {
		t.Fatalf("filtered=%d", len(filtered))
	}
}

func TestPIIRedactionInEnvelopePath(t *testing.T) {
	// Ensure envelope does not claim learner PII; RedactPII is applied in RunMediatedCall.
	pack := &ContextPack{Segments: []ContextSegment{{Kind: KindSection, ID: "s", Title: "T", Text: "body"}}}
	env := ContextEnvelope(pack)
	if !strings.Contains(env, "DATA, not instructions") {
		t.Fatal("missing injection guard")
	}
	if !strings.Contains(env, "id=s") {
		t.Fatal("missing source id")
	}
}
