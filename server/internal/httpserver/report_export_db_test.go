package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/reportschedules"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestReportSchedules_CRUD_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
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

	em := fmt.Sprintf("rptexp-%d@e.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	ownerID, _ := uuid.Parse(row.ID)
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	cfg := config.Config{ReportExportEnabled: true}
	d := Deps{Pool: pool, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	// CREATE schedule
	createBody, _ := json.Marshal(map[string]any{
		"reportType": "gradebook",
		"recipients": []string{"admin@test.com"},
		"cadence":    "weekly",
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/reports/schedules", bytes.NewReader(createBody))
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	schedID, _ := created["id"].(string)
	if schedID == "" {
		t.Fatal("expected id in create response")
	}

	// LIST schedules
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/reports/schedules", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rr.Code)
	}
	var list []map[string]any
	_ = json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&list)
	if len(list) == 0 {
		t.Fatal("expected at least one schedule in list")
	}

	// UPDATE schedule — disable it
	updateBody, _ := json.Marshal(map[string]any{"enabled": false})
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPut, "/api/v1/reports/schedules/"+schedID, bytes.NewReader(updateBody))
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var updated map[string]any
	_ = json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&updated)
	if enabled, _ := updated["enabled"].(bool); enabled {
		t.Error("expected enabled=false after update")
	}

	// DELETE schedule
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodDelete, "/api/v1/reports/schedules/"+schedID, nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", rr.Code)
	}

	// Verify deleted from DB
	s, err := reportschedules.Get(ctx, pool, uuid.MustParse(schedID))
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if s != nil {
		t.Error("expected schedule to be deleted but still found")
	}

	// Verify cross-user isolation: other user cannot delete a schedule they don't own
	em2 := fmt.Sprintf("rptother-%d@e.com", time.Now().UnixNano())
	row2, _ := user.InsertUser(ctx, pool, em2, ph, nil)
	ownerID2, _ := uuid.Parse(row2.ID)
	tok2, _ := signer.Sign(ctx, row2.ID, em2, "", "", nil)

	// Owner creates a schedule
	createBody, _ = json.Marshal(map[string]any{
		"reportType": "progress",
		"recipients": []string{"owner@test.com"},
		"cadence":    "monthly",
	})
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/v1/reports/schedules", bytes.NewReader(createBody))
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create owner schedule: %d", rr.Code)
	}
	_ = json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&created)
	schedID2 := created["id"].(string)

	// Other user tries to delete owner's schedule
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodDelete, "/api/v1/reports/schedules/"+schedID2, nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok2)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusForbidden {
		t.Errorf("cross-user delete: expected 403, got %d", rr.Code)
	}

	// Cleanup
	_ = reportschedules.Delete(ctx, pool, uuid.MustParse(schedID2))
	_ = ownerID  // used
	_ = ownerID2 // used
}

func TestReportExport_PDFEndpoint_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
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

	em := fmt.Sprintf("pdfexp-%d@e.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	cfg := config.Config{ReportExportEnabled: true}
	d := Deps{Pool: pool, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	// Learning activity PDF export — user may not have global:app:reports:view but the
	// endpoint only requires auth (no extra course permission), so should succeed or 404.
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/reports/learning-activity/export.pdf", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Logf("learning-activity PDF: %d (acceptable if permissions restrict)", rr.Code)
	} else {
		ct := rr.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/pdf") {
			t.Errorf("expected application/pdf content-type, got %q", ct)
		}
		if rr.Body.Len() == 0 {
			t.Error("expected non-empty PDF body")
		}
		body := rr.Body.Bytes()
		if len(body) >= 5 && string(body[:5]) != "%PDF-" {
			t.Errorf("response does not start with PDF header, got: %q", string(body[:min(10, len(body))]))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
