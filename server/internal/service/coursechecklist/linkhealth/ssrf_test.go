package linkhealth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateURLBlocksPrivateRanges(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/admin",
		"http://10.0.0.5/x",
		"http://172.16.1.1/x",
		"http://192.168.1.1/x",
		"http://169.254.169.254/latest/meta-data/",
		"http://localhost/x",
		"http://[::1]/",
	}
	for _, raw := range cases {
		_, reason, err := ValidateURL(raw)
		if err == nil {
			t.Fatalf("%s: expected block, got nil err", raw)
		}
		if reason != BlockedPrivateRange && reason != BlockedInvalidURL {
			t.Fatalf("%s: reason=%s", raw, reason)
		}
	}
}

func TestCheckOneBlocksMetadataURL(t *testing.T) {
	c := &Checker{Now: func() time.Time { return time.Unix(0, 0).UTC() }}
	res := c.checkOne(t.Context(), "http://169.254.169.254/latest/meta-data/")
	if res.Result != ResultSkipped {
		t.Fatalf("result=%s want skipped", res.Result)
	}
	if res.Reason != string(BlockedPrivateRange) {
		t.Fatalf("reason=%s", res.Reason)
	}
}

func TestRedirectToLoopbackAborted(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(final.Close)

	// Intermediate redirects to 127.0.0.1 — CheckRedirect must abort.
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1/", http.StatusFound)
	}))
	t.Cleanup(redirector.Close)

	c := &Checker{
		Client: &http.Client{
			Timeout: PerRequestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if reason, blocked := IsBlockedHost(req.URL.Hostname()); blocked {
					return &blockError{reason: reason}
				}
				return nil
			},
		},
	}
	res := c.checkOne(t.Context(), redirector.URL+"/")
	if res.Result != ResultSkipped && res.Result != ResultError {
		t.Fatalf("result=%s reason=%s", res.Result, res.Reason)
	}
	if res.Result == ResultOK {
		t.Fatal("must not succeed when redirect targets loopback")
	}
}

type blockError struct{ reason BlockedReason }

func (e *blockError) Error() string { return "redirect blocked: " + string(e.reason) }

func TestCapAt200URLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	urls := make([]string, 0, 300)
	for i := 0; i < 300; i++ {
		urls = append(urls, srv.URL+"/"+strings.Repeat("a", i%10)+string(rune('a'+i%26)))
	}
	// Make them unique with query params.
	for i := range urls {
		urls[i] = srv.URL + "/p?i=" + itoa(i)
	}
	c := &Checker{Now: time.Now}
	results := c.CheckURLs(t.Context(), urls)
	skipped := 0
	checked := 0
	for _, r := range results {
		if r.Result == ResultSkipped && r.Reason == "cap" {
			skipped++
		} else {
			checked++
		}
	}
	if checked > MaxURLsPerCourse {
		t.Fatalf("checked %d > %d", checked, MaxURLsPerCourse)
	}
	if skipped < 100 {
		t.Fatalf("expected ~100 capped skips, got %d (checked=%d total=%d)", skipped, checked, len(results))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func TestNoCookiesOrAuthSent(t *testing.T) {
	var sawCookie, sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			sawCookie = true
		}
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &Checker{}
	_ = c.checkOne(t.Context(), srv.URL)
	if sawCookie || sawAuth {
		t.Fatalf("cookie=%v auth=%v", sawCookie, sawAuth)
	}
}
