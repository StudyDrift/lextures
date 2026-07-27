package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchURL_notModified(t *testing.T) {
	var extracted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("User-agent: *\nDisallow:\n"))
			return
		}
		if r.Header.Get("If-None-Match") == `"abc"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		extracted = true
		w.Header().Set("ETag", `"abc"`)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><main>Hello cache</main></body></html>`))
	}))
	t.Cleanup(srv.Close)

	client := srv.Client()
	out, err := FetchURLWithClient(t.Context(), client, srv.URL+"/page", `"abc"`, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !out.NotModified {
		t.Fatalf("expected 304, got %#v", out)
	}
	if extracted {
		t.Fatal("body should not be read/extracted on 304")
	}

	st, reason := ReasonForFetchError(ErrSSRFBlocked)
	if st != StatusBlocked || reason != "blocked: private network" {
		t.Fatalf("%s %s", st, reason)
	}
	st, reason = ReasonForFetchError(ErrTooLarge)
	if st != StatusUnsupported || reason != "unsupported: exceeds size cap" {
		t.Fatalf("%s %s", st, reason)
	}
}

func TestFetchURL_sizeCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nDisallow:\n"))
			return
		}
		w.Header().Set("Content-Length", "31457280") // 30 MB
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	_, err := FetchURLWithClient(t.Context(), srv.Client(), srv.URL+"/big.pdf", "", "", false)
	if err != ErrTooLarge {
		t.Fatalf("got %v", err)
	}
}

func TestIngestPolicy(t *testing.T) {
	p := IngestPolicy{Mode: IngestOff}
	if err := p.Allow("https://example.com"); err != ErrIngestDisabled {
		t.Fatalf("got %v", err)
	}
	p = IngestPolicy{Mode: IngestAllowlist, Allowlist: []string{"ok.example"}}
	if err := p.Allow("https://nope.example/"); err != ErrHostNotAllowlisted {
		t.Fatalf("got %v", err)
	}
	if err := p.Allow("https://ok.example/x"); err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestBudgetErrorUnwrap(t *testing.T) {
	e := &BudgetError{Level: "monthly_course", Message: "cap"}
	if !errorsIs(e, ErrBudgetCourseMonthly) {
		t.Fatal("unwrap")
	}
}

func errorsIs(err, target error) bool {
	type unwrapper interface{ Unwrap() error }
	for err != nil {
		if err == target {
			return true
		}
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
