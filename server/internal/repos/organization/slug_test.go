package organization

import "testing"

func TestSuggestSlugFromName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Chase's Org":  "chase-s-org",
		"Riverdale USD": "riverdale-usd",
		"  ACME  ":      "acme",
	}
	for in, want := range cases {
		if got := SuggestSlugFromName(in); got != want {
			t.Fatalf("SuggestSlugFromName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	t.Parallel()
	ok := []string{"chase", "riverdale-usd", "org2"}
	for _, s := range ok {
		if err := ValidateSlug(s); err != nil {
			t.Fatalf("ValidateSlug(%q): %v", s, err)
		}
	}
	bad := []string{"", "a", "chase_", "default", "login", "bad slug", "-start"}
	for _, s := range bad {
		if err := ValidateSlug(s); err == nil {
			t.Fatalf("ValidateSlug(%q): want error", s)
		}
	}
}