package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func TestContentTools_KillSwitch_Env_NoDB(t *testing.T) {
	t.Setenv(ctsvc.EnvKillSwitch, "on")
	if ctsvc.AvailableForCourse(true) {
		t.Fatal("AvailableForCourse should be false when kill-switch engaged")
	}

	d := Deps{}
	r := chi.NewRouter()
	d.registerContentToolsRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/content-tools/catalog", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("unexpected 200")
	}
}

func TestContentTools_ActiveForCourse_Helper(t *testing.T) {
	t.Setenv(ctsvc.EnvKillSwitch, "")
	if !ctsvc.ActiveForCourse(true) {
		t.Fatal("expected active")
	}
	if ctsvc.ActiveForCourse(false) {
		t.Fatal("flag off should be inactive")
	}
}

func TestContentTools_BuiltinRegistryLoads(t *testing.T) {
	reg := ctsvc.MustDefault()
	if reg.Get("noop_probe") == nil {
		t.Fatal("noop_probe must be registered")
	}
}

func TestContentTools_AuthoringRoutesRegistered_NoDB(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	d.registerContentToolsRoutes(r)

	for _, path := range []struct {
		method string
		url    string
	}{
		{http.MethodPost, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/duplicate"},
		{http.MethodGet, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/usage"},
		{http.MethodPost, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/submit"},
		{http.MethodPost, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/actions/grade"},
		{http.MethodGet, "/api/v1/courses/demo/content-tools/context/sources?itemId=00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/courses/demo/content-tools/context/preview"},
		{http.MethodPost, "/api/v1/courses/demo/content-tools/context/sources/00000000-0000-0000-0000-000000000001/reingest"},
		{http.MethodPatch, "/api/v1/courses/demo/content-tools/context/sources/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/analytics"},
		{http.MethodGet, "/api/v1/courses/demo/content-tools/analytics?itemId=00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/courses/demo/content-tools/analytics/course"},
		{http.MethodGet, "/api/v1/courses/demo/content-tools/my-progress?itemId=00000000-0000-0000-0000-000000000001"},
		{http.MethodPut, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/grade-link"},
		{http.MethodDelete, "/api/v1/courses/demo/content-tools/instances/00000000-0000-0000-0000-000000000001/grade-link"},
	} {
		req := httptest.NewRequest(path.method, path.url, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound && rr.Body.String() == "404 page not found\n" {
			t.Fatalf("%s %s: chi returned 404 page not found (route missing)", path.method, path.url)
		}
	}
}

func TestContentTools_RuntimeReadonly_Env(t *testing.T) {
	t.Setenv(ctsvc.EnvRuntimeReadonly, "on")
	if !ctsvc.RuntimeReadonly() {
		t.Fatal("expected runtime readonly")
	}
}

func TestContentTools_DuplicateRateLimit_NoDB(t *testing.T) {
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	contentToolsDupRateMu.Lock()
	delete(contentToolsDupRateByUser, uid)
	contentToolsDupRateMu.Unlock()

	for i := 0; i < contentToolsDuplicateRateLimitPerMin; i++ {
		if !checkContentToolsDuplicateRateLimit(uid) {
			t.Fatalf("request %d unexpectedly rate limited", i+1)
		}
	}
	if checkContentToolsDuplicateRateLimit(uid) {
		t.Fatal("expected rate limit after burst")
	}
}
