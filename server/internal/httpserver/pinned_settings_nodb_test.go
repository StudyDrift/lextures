package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/platformstate"
)

func pinnedSettingsDeps(enabled bool) Deps {
	cfg := config.Config{FFPinnedSettings: enabled}
	return Deps{
		Config:    cfg,
		Platform:  platformstate.New(cfg),
		JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"),
	}
}

func TestPinnedSettings_GetUnauthorized(t *testing.T) {
	d := pinnedSettingsDeps(true)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/pinned-settings", nil)
	rec := httptest.NewRecorder()
	d.handleGetMyPinnedSettings()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestPinnedSettings_PutUnauthorized(t *testing.T) {
	d := pinnedSettingsDeps(true)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/pinned-settings/quiz",
		strings.NewReader(`{"settingKeys":["quiz.scheduling.due-date"]}`))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("surface", "quiz")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	d.handlePutMyPinnedSettings()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestPinnedSettings_FlagOff_404(t *testing.T) {
	d := pinnedSettingsDeps(false)
	rec := httptest.NewRecorder()
	if d.requirePinnedSettings(rec) {
		t.Fatal("expected require to fail when flag off")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj, _ := body["error"].(map[string]any)
	if msg, _ := errObj["message"].(string); msg != "Pinned settings are not enabled." {
		t.Fatalf("message=%q", msg)
	}
}

func TestPinnedSettings_FlagOn_RequirePasses(t *testing.T) {
	d := pinnedSettingsDeps(true)
	rec := httptest.NewRecorder()
	if !d.requirePinnedSettings(rec) {
		t.Fatal("expected flag-on require to pass")
	}
}

func TestPinnedSettings_RateLimitBucket(t *testing.T) {
	d := pinnedSettingsDeps(true)
	uid := uuid.MustParse("b0000000-0000-4000-8000-000000000042")
	// Isolate from other tests: clear any prior entry.
	pinnedSettingsRateMu.Lock()
	delete(pinnedSettingsRateByUser, uid)
	pinnedSettingsRateMu.Unlock()

	for i := 0; i < pinnedSettingsWriteLimitPerMinute; i++ {
		if !d.checkPinnedSettingsWriteRate(uid) {
			t.Fatalf("request %d unexpectedly rate limited", i+1)
		}
	}
	if d.checkPinnedSettingsWriteRate(uid) {
		t.Fatal("expected 61st write to be rate limited")
	}
}
