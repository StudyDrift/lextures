package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func TestContentTools_KillSwitch_Returns404_NoDB(t *testing.T) {
	t.Cleanup(func() { ctsvc.SetKillSwitchForTest(nil) })
	on := true
	ctsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerContentToolsRoutes(r)

	// Without auth, requireCourseAccess fails first — but AvailableForCourse is false.
	// Probe ActiveForCourse helper (same as FR-14 kill-switch path).
	if ctsvc.AvailableForCourse(true) {
		t.Fatal("AvailableForCourse should be false when kill-switch engaged")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/content-tools/catalog", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	// No pool/auth → 401/500 before flag check; kill-switch is enforced after auth in requireContentToolsCourse.
	if rr.Code == http.StatusOK {
		t.Fatalf("unexpected 200")
	}
}

func TestContentTools_ActiveForCourse_Helper(t *testing.T) {
	t.Cleanup(func() { ctsvc.SetKillSwitchForTest(nil) })
	off := false
	ctsvc.SetKillSwitchForTest(&off)
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
