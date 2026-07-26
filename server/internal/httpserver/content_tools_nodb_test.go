package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

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
