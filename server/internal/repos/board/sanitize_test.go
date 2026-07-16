package board

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizePostHTML_stripsScript(t *testing.T) {
	t.Parallel()
	got := SanitizePostHTML(`<p>Hi</p><script>alert(1)</script><strong>ok</strong>`)
	if strings.Contains(got, "script") || strings.Contains(got, "alert") {
		t.Fatalf("script not stripped: %q", got)
	}
	if !strings.Contains(got, "strong") && !strings.Contains(got, "ok") {
		t.Fatalf("expected safe markup kept: %q", got)
	}
}

func TestNormalizeBody_html(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]string{"html": `<p>a</p><img src=x onerror=alert(1)>`})
	out, err := NormalizeBody(raw)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	html, _ := obj["html"].(string)
	if strings.Contains(html, "onerror") || strings.Contains(html, "img") {
		t.Fatalf("unsafe attrs/tags kept: %q", html)
	}
}
