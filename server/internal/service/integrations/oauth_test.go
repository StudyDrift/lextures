package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "the-code" {
			t.Errorf("code = %q", r.Form.Get("code"))
		}
		if r.Form.Get("client_secret") != "secret" {
			t.Errorf("client_secret = %q", r.Form.Get("client_secret"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":3600,"scope":"a b"}`))
	}))
	defer srv.Close()

	s := testService()
	s.HTTP = srv.Client()
	meta := s.Providers[ProviderGoogleClassroom]
	meta.TokenURL = srv.URL
	s.Providers[ProviderGoogleClassroom] = meta

	tokens, err := s.exchangeCode(context.Background(), ProviderGoogleClassroom, "the-code")
	if err != nil {
		t.Fatalf("exchangeCode error: %v", err)
	}
	if tokens.AccessToken != "at" || tokens.RefreshToken != "rt" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}
	if tokens.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if len(tokens.Scopes) != 2 {
		t.Errorf("scopes = %v", tokens.Scopes)
	}
}

func TestExchangeCodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	s := testService()
	s.HTTP = srv.Client()
	meta := s.Providers[ProviderGoogleClassroom]
	meta.TokenURL = srv.URL
	s.Providers[ProviderGoogleClassroom] = meta
	if _, err := s.exchangeCode(context.Background(), ProviderGoogleClassroom, "x"); err == nil {
		t.Error("expected error on non-2xx token response")
	}
}

func TestResolveAccountHTTPNonGoogleDeterministic(t *testing.T) {
	s := testService()
	a, err := s.resolveAccountHTTP(context.Background(), ProviderCanva, Tokens{AccessToken: "tok"})
	if err != nil {
		t.Fatalf("resolveAccountHTTP error: %v", err)
	}
	b, _ := s.resolveAccountHTTP(context.Background(), ProviderCanva, Tokens{AccessToken: "tok"})
	if a == "" || a != b {
		t.Errorf("expected stable non-empty id, got %q and %q", a, b)
	}
}

func TestHTTPClassroomClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/courses/c1/students":
			_, _ = w.Write([]byte(`{"students":[{"userId":"u1","profile":{"name":{"fullName":"Stu"},"emailAddress":"stu@example.com"}}]}`))
		case "/courses/c1/teachers":
			_, _ = w.Write([]byte(`{"teachers":[{"userId":"t1","profile":{"name":{"fullName":"Teach"},"emailAddress":"teach@example.com"}}]}`))
		case "/courses/c1/courseWork":
			_, _ = w.Write([]byte(`{"courseWork":[{"id":"a1","title":"HW1","maxPoints":100,"dueDate":{"year":2026,"month":6,"day":1}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &httpClassroomClient{http: srv.Client(), baseURL: srv.URL}
	ctx := context.Background()

	members, err := c.ListMembers(ctx, "tok", "c1")
	if err != nil {
		t.Fatalf("ListMembers error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}
	if members[0].Role != "student" || members[1].Role != "teacher" {
		t.Errorf("unexpected roles: %+v", members)
	}

	work, err := c.ListCourseWork(ctx, "tok", "c1")
	if err != nil {
		t.Fatalf("ListCourseWork error: %v", err)
	}
	if len(work) != 1 || work[0].Title != "HW1" || work[0].MaxPoints != 100 {
		t.Fatalf("unexpected coursework: %+v", work)
	}
	if work[0].DueDate == nil || work[0].DueDate.Year() != 2026 {
		t.Errorf("due date not parsed: %+v", work[0].DueDate)
	}
}

func TestFreshAccessTokenNotExpired(t *testing.T) {
	// When the stored token is far from expiry, freshAccessToken must return it
	// without hitting the network (HTTP client left nil to prove no call).
	s := testService()
	future := s.now().Add(time.Hour)
	access, err := encryptForTest("live-token")
	if err != nil {
		t.Fatal(err)
	}
	refresh, _ := encryptForTest("refresh")
	conn := connForTest(access, refresh, &future)
	got, err := s.freshAccessToken(context.Background(), conn)
	if err != nil {
		t.Fatalf("freshAccessToken error: %v", err)
	}
	if got != "live-token" {
		t.Errorf("token = %q, want live-token", got)
	}
}
