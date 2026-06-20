package integrations

import (
	"testing"
	"time"
)

func TestGenerateLTILaunchParams(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	base := LTILaunch{
		LaunchURL:      "https://quizlet.com/lti/launch",
		ConsumerKey:    "key123",
		ConsumerSecret: "shh",
		ResourceLinkID: "item-1",
		ContextID:      "course-1",
		ContextTitle:   "Algebra",
		UserID:         "user-1",
		Roles:          "Learner",
		UserEmail:      "s@example.com",
		UserName:       "Stu Dent",
	}
	p, err := GenerateLTILaunchParams(base, now, "nonce-abc")
	if err != nil {
		t.Fatalf("GenerateLTILaunchParams error: %v", err)
	}
	for _, k := range []string{
		"lti_message_type", "lti_version", "oauth_consumer_key",
		"oauth_signature", "oauth_signature_method", "oauth_timestamp", "oauth_nonce",
	} {
		if p[k] == "" {
			t.Errorf("missing required launch param %q", k)
		}
	}
	if p["oauth_signature_method"] != "HMAC-SHA1" {
		t.Errorf("signature method = %q, want HMAC-SHA1", p["oauth_signature_method"])
	}
	if p["oauth_timestamp"] != "1700000000" {
		t.Errorf("timestamp = %q", p["oauth_timestamp"])
	}

	// Determinism: identical inputs produce an identical signature.
	p2, _ := GenerateLTILaunchParams(base, now, "nonce-abc")
	if p["oauth_signature"] != p2["oauth_signature"] {
		t.Error("signature should be deterministic for identical inputs")
	}

	// A different secret must change the signature.
	alt := base
	alt.ConsumerSecret = "different"
	p3, _ := GenerateLTILaunchParams(alt, now, "nonce-abc")
	if p["oauth_signature"] == p3["oauth_signature"] {
		t.Error("signature should change when the secret changes")
	}
}

func TestGenerateLTILaunchParamsValidation(t *testing.T) {
	if _, err := GenerateLTILaunchParams(LTILaunch{}, time.Now(), "n"); err == nil {
		t.Error("expected error for missing url/key/secret")
	}
}

func TestOAuthEscape(t *testing.T) {
	cases := map[string]string{
		"a b":             "a%20b",
		"AZaz09-._~":      "AZaz09-._~",
		"name@host.com":   "name%40host.com",
		"https://x/y?z=1": "https%3A%2F%2Fx%2Fy%3Fz%3D1",
	}
	for in, want := range cases {
		if got := oauthEscape(in); got != want {
			t.Errorf("oauthEscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"HTTPS://Quizlet.com:443/lti?x=1": "https://quizlet.com/lti",
		"http://Example.com:80/a":         "http://example.com/a",
		"https://host.com/path":           "https://host.com/path",
	}
	for in, want := range cases {
		if got := normalizeURL(in); got != want {
			t.Errorf("normalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}
