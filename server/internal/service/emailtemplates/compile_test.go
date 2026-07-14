package emailtemplates

import (
	"strings"
	"testing"
)

func TestCompile_formattingAndTokenPreservation(t *testing.T) {
	html, err := Compile("**Hi** [x]({{link}})")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>Hi</strong>") {
		if !strings.Contains(html, "Hi") || !strings.Contains(html, "strong") {
			t.Fatalf("expected strong Hi, got %q", html)
		}
	}
	if !strings.Contains(html, `href="{{link}}"`) && !strings.Contains(html, "{{link}}") {
		t.Fatalf("expected {{link}} preserved, got %q", html)
	}
	if !strings.Contains(html, "<a ") {
		t.Fatalf("expected anchor, got %q", html)
	}
}

func TestCompile_stripsScriptAndUnsafe(t *testing.T) {
	html, err := Compile("Hello <script>alert(1)</script> **ok**\n\n[bad](javascript:alert(1))")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<script") {
		t.Fatalf("script tag leaked: %q", html)
	}
	if strings.Contains(strings.ToLower(html), "javascript:") {
		t.Fatalf("javascript: href leaked: %q", html)
	}
	// iframe raw HTML in markdown (unsafe disabled) should not survive.
	html2, err := Compile(`click <iframe src="https://evil.test"></iframe>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html2, "<iframe") {
		t.Fatalf("iframe leaked: %q", html2)
	}
}

func TestCompile_gfmListAndHeading(t *testing.T) {
	html, err := Compile("## Title\n\n- one\n- two\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<h2") {
		t.Fatalf("expected h2, got %q", html)
	}
	if !strings.Contains(html, "<li>") {
		t.Fatalf("expected list items, got %q", html)
	}
}

func TestCompile_emptyRejected(t *testing.T) {
	if _, err := Compile("   "); err == nil {
		t.Fatal("expected error for empty markdown")
	}
}

func TestCompile_tableAllowed(t *testing.T) {
	md := `| A | B |
| --- | --- |
| 1 | 2 |
`
	html, err := Compile(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<table") {
		t.Fatalf("expected table, got %q", html)
	}
}
