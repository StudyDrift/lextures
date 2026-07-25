package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

func TestAdaptiveContent_KillSwitch_Mutations503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{} // no pool — handlers return 503 before DB for kill-switch mutations
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	// PUT settings should 503 without needing DB.
	req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/demo/adaptive-content/settings", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	// Without auth, requireCourseItemCreate runs after kill-switch... wait, kill-switch is first.
	// Actually in handleAdaptiveContentSettingsPut, kill-switch is checked before auth.
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("PUT settings: status %d body %s", rr.Code, rr.Body.String())
	}
	var body apierr.Body
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != apierr.CodeServiceUnavailable {
		t.Fatalf("code: %q", body.Error.Code)
	}

	// GET settings does not hit kill-switch first — it requires access (no pool → will fail later).
	// Verify kill-switch path is not applied by checking ActiveForCourse.
	if acsvc.ActiveForCourse(true) {
		t.Fatal("ActiveForCourse should be false when kill-switch engaged")
	}
}

func TestAdaptiveContent_ActiveForCourse_Helper(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	off := false
	acsvc.SetKillSwitchForTest(&off)
	if !acsvc.ActiveForCourse(true) {
		t.Fatal("expected active")
	}
}

func TestAdaptiveContent_VariantPreview_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/units/"+uuidZero+"/variants/preview", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

// uuidZero is a valid path uuid that will not be reached past kill-switch.
const uuidZero = "00000000-0000-0000-0000-000000000000"

func TestAdaptiveContent_Prewarm_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/units/"+uuidZero+"/prewarm", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_ViewedOriginal_NoAuth401or4xx_NoDB(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/units/"+uuidZero+"/viewed-original", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	// Without auth middleware wired, requireCourseAccess fails (401/403/500 depending on Deps).
	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-OK without auth, got %d", rr.Code)
	}
}

func TestAdaptiveContent_OptoutGet_NoAuth_NoDB(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/adaptive-content/optout", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-OK without auth, got %d", rr.Code)
	}
}

func TestAdaptiveContent_SettingsPatch_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/demo/adaptive-content/settings", bytes.NewReader([]byte(`{"generationPaused":true}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_AdminAdaptiveContent_Unauthorized_NoDB(t *testing.T) {
	d := Deps{} // no JWT
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/adaptive-content", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_VariantApprove_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/variants/"+uuidZero+"/approve", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_VariantRevoke_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/variants/"+uuidZero+"/revoke", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_Bulk_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/units/"+uuidZero+"/variants/bulk", bytes.NewReader([]byte(`{"action":"approve","variantIds":[]}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_EffectivenessRefresh_KillSwitch503_NoDB(t *testing.T) {
	t.Cleanup(func() { acsvc.SetKillSwitchForTest(nil) })
	on := true
	acsvc.SetKillSwitchForTest(&on)

	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/effectiveness/refresh", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAdaptiveContent_ContestRoute_Registered_NoDB(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/adaptive-content/units/"+uuidZero+"/contest", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	// Without auth/DB we expect 401/403/500-class, not 404.
	if rr.Code == http.StatusNotFound {
		t.Fatalf("contest route not registered: %d", rr.Code)
	}
}

func TestAdaptiveContent_OversightRoute_Registered_NoDB(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	d.registerAdaptiveContentRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/adaptive-content/oversight", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("oversight route not registered: %d", rr.Code)
	}
}
