package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/platformstate"
	"github.com/lextures/lextures/server/internal/repos/pinnedsettings"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestPinnedSettings_AuthenticatedHandlersPg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	em := "pinned-" + time.Now().Format("20060102150405.999999999") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	userID := uuid.MustParse(row.ID)

	cfg := config.Config{FFPinnedSettings: true}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	d := Deps{
		Pool:      pool,
		Config:    cfg,
		Platform:  platformstate.New(cfg),
		JWTSigner: signer,
	}

	// AC-1: empty GET
	{
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodGet, "/api/v1/me/pinned-settings", nil)
		w := httptest.NewRecorder()
		d.handleGetMyPinnedSettings()(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("GET empty: status=%d body=%s", w.Code, w.Body.String())
		}
		var resp pinnedSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Surfaces.Assignment) != 0 || len(resp.Surfaces.Quiz) != 0 {
			t.Fatalf("want empty surfaces, got %+v", resp.Surfaces)
		}
	}

	// AC-2: PUT quiz keys, GET preserves order
	keys := []string{"quiz.presentation.lockdown-mode", "quiz.scheduling.due-date"}
	{
		body, _ := json.Marshal(map[string]any{"settingKeys": keys})
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/quiz", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "quiz")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT quiz: status=%d body=%s", w.Code, w.Body.String())
		}
		var resp pinnedSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if strings.Join(resp.Surfaces.Quiz, ",") != strings.Join(keys, ",") {
			t.Fatalf("PUT response quiz=%v want %v", resp.Surfaces.Quiz, keys)
		}
	}
	{
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodGet, "/api/v1/me/pinned-settings", nil)
		w := httptest.NewRecorder()
		d.handleGetMyPinnedSettings()(w, r)
		var resp pinnedSettingsResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if strings.Join(resp.Surfaces.Quiz, ",") != strings.Join(keys, ",") {
			t.Fatalf("GET after PUT quiz=%v", resp.Surfaces.Quiz)
		}
	}

	// AC-3: reorder replace
	reordered := []string{"quiz.scheduling.due-date", "quiz.presentation.lockdown-mode"}
	{
		body, _ := json.Marshal(map[string]any{"settingKeys": reordered})
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/quiz", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "quiz")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("reorder: %d %s", w.Code, w.Body.String())
		}
		var resp pinnedSettingsResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if strings.Join(resp.Surfaces.Quiz, ",") != strings.Join(reordered, ",") {
			t.Fatalf("reorder got %v", resp.Surfaces.Quiz)
		}
	}

	// AC-4: 13 keys → 400, stored unchanged
	{
		tooMany := make([]string, 13)
		for i := range tooMany {
			tooMany[i] = fmt.Sprintf("quiz.key.%d", i)
		}
		body, _ := json.Marshal(map[string]any{"settingKeys": tooMany})
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/quiz", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "quiz")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("13 keys: status=%d", w.Code)
		}
		got, err := pinnedsettings.Get(ctx, pool, userID, "quiz")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, ",") != strings.Join(reordered, ",") {
			t.Fatalf("stored after reject=%v", got)
		}
	}

	// AC-5 shape
	{
		body := []byte(`{"settingKeys":["Quiz.Bad Key!"]}`)
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/quiz", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "quiz")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad key: %d", w.Code)
		}
	}

	// AC-6 duplicate
	{
		body := []byte(`{"settingKeys":["quiz.a.b","quiz.a.b"]}`)
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/quiz", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "quiz")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("dup: %d", w.Code)
		}
	}

	// AC-10 bad surface → 400 not 404
	{
		body := []byte(`{"settingKeys":[]}`)
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/discussion", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "discussion")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad surface: %d want 400", w.Code)
		}
	}

	// AC-11 empty array clear
	{
		body := []byte(`{"settingKeys":[]}`)
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodPut, "/api/v1/me/pinned-settings/quiz", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("surface", "quiz")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		d.handlePutMyPinnedSettings()(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("clear: %d %s", w.Code, w.Body.String())
		}
		var resp pinnedSettingsResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Surfaces.Quiz) != 0 {
			t.Fatalf("clear response quiz=%v", resp.Surfaces.Quiz)
		}
	}

	// AC-7 flag off
	{
		dOff := Deps{
			Pool:      pool,
			Config:    config.Config{FFPinnedSettings: false},
			Platform:  platformstate.New(config.Config{FFPinnedSettings: false}),
			JWTSigner: signer,
		}
		r := bearerRequest(t, ctx, signer, row.ID, em, http.MethodGet, "/api/v1/me/pinned-settings", nil)
		w := httptest.NewRecorder()
		dOff.handleGetMyPinnedSettings()(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("flag off: %d", w.Code)
		}
	}

	// AC-9 cascade: delete user removes rows
	{
		if _, err := pool.Exec(ctx, `
INSERT INTO settings.user_pinned_settings (user_id, surface, setting_keys)
VALUES ($1, 'assignment', ARRAY['assignment.scheduling.due-date'])
ON CONFLICT (user_id, surface) DO UPDATE SET setting_keys = EXCLUDED.setting_keys
`, userID); err != nil {
			t.Fatalf("seed pins: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID); err != nil {
			t.Fatalf("delete user: %v", err)
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM settings.user_pinned_settings WHERE user_id = $1`, userID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("cascade: want 0 rows, got %d", n)
		}
	}
}
