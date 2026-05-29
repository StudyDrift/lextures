package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/platformstate"
)

func readAloudDeps(enabled bool) Deps {
	cfg := config.Config{
		ReadAloudEnabled: enabled,
		FFReadAloud:      enabled,
	}
	return Deps{
		Config:    cfg,
		Platform:  platformstate.New(cfg),
		JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"),
	}
}

func bearerRequest(t *testing.T, d Deps, method, path string, body []byte) *http.Request {
	t.Helper()
	tok, err := d.JWTSigner.Sign(context.Background(), "a0000000-0000-4000-8000-000000000099", "tts@test.invalid", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	}
	r.Header.Set("Authorization", "Bearer "+tok)
	return r
}

func TestReadAloud_GetPreferencesUnauthorized(t *testing.T) {
	d := readAloudDeps(true)
	w := httptest.NewRecorder()
	d.handleGetMyReadingPreferences()(w, httptest.NewRequest(http.MethodGet, "/api/v1/me/reading-preferences", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestReadAloud_TTSSynthesizeFeatureOff(t *testing.T) {
	d := readAloudDeps(false)
	w := httptest.NewRecorder()
	r := bearerRequest(t, d, http.MethodPost, "/api/v1/tts/synthesize", []byte(`{"text":"Hi","lang":"en","speed":1}`))
	d.handlePostTTSSynthesize()(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestReadAloud_AuthenticatedHandlersPg(t *testing.T) {
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

	cfg := config.Config{ReadAloudEnabled: true, FFReadAloud: true}
	d := Deps{
		Pool:      pool,
		Config:    cfg,
		Platform:  platformstate.New(cfg),
		JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"),
	}

	t.Run("patch invalid speed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := bearerRequest(t, d, http.MethodPatch, "/api/v1/me/reading-preferences", []byte(`{"ttsSpeed":3}`))
		d.handlePatchMyReadingPreferences()(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("tts synthesize wav", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := bearerRequest(t, d, http.MethodPost, "/api/v1/tts/synthesize", []byte(`{"text":"Hello world","lang":"en-US","speed":1}`))
		d.handlePostTTSSynthesize()(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		if w.Header().Get("Content-Type") != "audio/wav" {
			t.Fatalf("expected audio/wav content type")
		}
		if len(w.Body.Bytes()) < 44 {
			t.Fatalf("expected wav bytes")
		}
	})

	t.Run("preferences round trip", func(t *testing.T) {
		patchBody, _ := json.Marshal(map[string]any{"ttsEnabled": true, "ttsSpeed": 1.5})
		w := httptest.NewRecorder()
		r := bearerRequest(t, d, http.MethodPatch, "/api/v1/me/reading-preferences", patchBody)
		d.handlePatchMyReadingPreferences()(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("patch: %d %s", w.Code, w.Body.String())
		}

		w2 := httptest.NewRecorder()
		r2 := bearerRequest(t, d, http.MethodGet, "/api/v1/me/reading-preferences", nil)
		d.handleGetMyReadingPreferences()(w2, r2)
		if w2.Code != http.StatusOK {
			t.Fatalf("get: %d", w2.Code)
		}
		var out struct {
			TTSEnabled bool    `json:"ttsEnabled"`
			TTSSpeed   float64 `json:"ttsSpeed"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if !out.TTSEnabled || out.TTSSpeed != 1.5 {
			t.Fatalf("unexpected prefs: %+v", out)
		}
	})
}
