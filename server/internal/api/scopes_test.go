package api

import "testing"

func TestNormalizeScopes(t *testing.T) {
	t.Parallel()
	out, ok := NormalizeScopes([]string{"courses:read", " grades:read ", "courses:read", "bad:scope"})
	if ok || out != nil {
		t.Fatalf("expected invalid scope rejection, got ok=%v out=%v", ok, out)
	}
	out, ok = NormalizeScopes([]string{"mcp:connect", "courses:read", "courses:read"})
	if !ok || len(out) != 2 {
		t.Fatalf("expected deduped valid scopes, got ok=%v out=%v", ok, out)
	}
}

func TestAllScopesNonEmpty(t *testing.T) {
	t.Parallel()
	for _, s := range AllScopes() {
		if s.ID == "" || s.Label == "" || s.Group == "" {
			t.Fatalf("incomplete scope: %+v", s)
		}
	}
}
