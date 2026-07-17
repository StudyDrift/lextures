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
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func setupQuizKitTest(t *testing.T, ctx context.Context, role string, courseFlag, masterFlag bool) (
	*pgxpool.Pool, http.Handler, string, string, uuid.UUID,
) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}

	em := fmt.Sprintf("quizkits-%s-%d@test.com", role, time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		pool.Close()
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])

	var courseID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Live Quizzes Test', $2) RETURNING id`,
		cc, uid,
	).Scan(&courseID); err != nil {
		pool.Close()
		t.Fatalf("course: %v", err)
	}
	if courseFlag {
		if _, err := pool.Exec(ctx, `UPDATE course.courses SET interactive_quizzes_enabled = true WHERE id = $1`, courseID); err != nil {
			pool.Close()
			t.Fatalf("enable course flag: %v", err)
		}
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, $3)`,
		courseID, uid, role,
	); err != nil {
		pool.Close()
		t.Fatalf("enroll: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		pool.Close()
		t.Fatalf("begin: %v", err)
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, uid, courseID, cc); err != nil {
		_ = tx.Rollback(ctx)
		pool.Close()
		t.Fatalf("grants: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		pool.Close()
		t.Fatalf("commit: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, _ := signer.Sign(ctx, row.ID, em, "", "", nil)
	cfg := config.Config{
		FFInteractiveQuizzes: masterFlag,
		FFIqLiveHosting:      masterFlag,
		FFIqTeamMode:         masterFlag,
		FFIqStudentPaced:     masterFlag,
		FFIqHomework:         masterFlag,
		FFIqGradebookPush:    masterFlag,
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg})
	return pool, h, tok, cc, courseID
}

func setupQuizKitTestWithCfg(
	t *testing.T,
	ctx context.Context,
	role string,
	courseFlag, masterFlag bool,
	mutate func(*config.Config),
) (*pgxpool.Pool, http.Handler, string, string, uuid.UUID) {
	t.Helper()
	pool, _, tok, cc, courseID := setupQuizKitTest(t, ctx, role, courseFlag, masterFlag)
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	cfg := config.Config{
		FFInteractiveQuizzes: masterFlag,
		FFIqLiveHosting:      masterFlag,
		FFIqTeamMode:         masterFlag,
		FFIqStudentPaced:     masterFlag,
		FFIqHomework:         masterFlag,
		FFIqGradebookPush:    masterFlag,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	return pool, NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg}), tok, cc, courseID
}

func TestQuizKits_FeatureGate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", false, true)
	defer pool.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("gate: expected 404 when course flag off, got %d %s", rr.Code, rr.Body.String())
	}

	b, _ := json.Marshal(map[string]any{
		"notebookEnabled":             true,
		"feedEnabled":                 false,
		"calendarEnabled":             true,
		"questionBankEnabled":         false,
		"lockdownModeEnabled":         false,
		"discussionsEnabled":          false,
		"interactiveQuizzesEnabled":   true,
	})
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/features", bytes.NewReader(b))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("enable via features: %d %s", rr2.Code, rr2.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &body)
	if body["interactiveQuizzesEnabled"] != true {
		t.Fatalf("expected interactiveQuizzesEnabled=true, got %v", body["interactiveQuizzesEnabled"])
	}

	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("list after enable: %d %s", rr3.Code, rr3.Body.String())
	}
}

func TestQuizKits_MasterFlagOff_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, false)
	defer pool.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("master off: expected 404, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestQuizKits_FullCRUD_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	var kitID, slug1 string
	{
		b, _ := json.Marshal(map[string]any{"title": "Unit 1 Review", "description": "Warm-up kit", "tags": []string{"review"}})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		kitID, _ = body["id"].(string)
		slug1, _ = body["slug"].(string)
		if kitID == "" || slug1 == "" {
			t.Fatal("create: missing id/slug")
		}
		if body["title"] != "Unit 1 Review" {
			t.Fatalf("create: title %v", body["title"])
		}
		if body["status"] != "draft" {
			t.Fatalf("create: status %v", body["status"])
		}
	}

	// Same title → distinct slug (AC-5).
	{
		b, _ := json.Marshal(map[string]any{"title": "Unit 1 Review"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create duplicate title: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		slug2, _ := body["slug"].(string)
		if slug2 == "" || slug2 == slug1 {
			t.Fatalf("expected distinct slug, got %q vs %q", slug2, slug1)
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits?q=Unit", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		kits, _ := body["kits"].([]any)
		if len(kits) != 2 {
			t.Fatalf("list: expected 2, got %d", len(kits))
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: %d %s", rr.Code, rr.Body.String())
		}
	}

	{
		b, _ := json.Marshal(map[string]any{"title": "Unit 1 Renamed"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("rename: %d %s", rr.Code, rr.Body.String())
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/duplicate", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("duplicate: %d %s", rr.Code, rr.Body.String())
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/archive", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("archive: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		if body["archived"] != true {
			t.Fatalf("archive: expected archived=true, got %v", body["archived"])
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		kits, _ := body["kits"].([]any)
		// archived kit excluded; duplicate + other same-title kit remain
		if len(kits) < 2 {
			t.Fatalf("list after archive: expected >=2, got %d", len(kits))
		}
		for _, raw := range kits {
			m, _ := raw.(map[string]any)
			if m["id"] == kitID {
				t.Fatal("archived kit should not appear in default list")
			}
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/restore", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("restore: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		if body["archived"] != false {
			t.Fatalf("restore: expected archived=false, got %v", body["archived"])
		}
	}

	// Soft-archive is reversible and not hard-deleted.
	var stillExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM quizgame.kits WHERE id = $1)`, kitID).Scan(&stillExists); err != nil || !stillExists {
		t.Fatalf("kit should still exist after archive/restore: exists=%v err=%v", stillExists, err)
	}

	// Course delete cascades kits.
	if _, err := pool.Exec(ctx, `DELETE FROM course.courses WHERE id = $1`, courseID); err != nil {
		t.Fatalf("delete course: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM quizgame.kits WHERE course_id = $1`, courseID).Scan(&count); err != nil {
		t.Fatalf("count kits: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cascade delete, got %d kits", count)
	}
}

func TestQuizKits_StudentCannotCreate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "student", true, true)
	defer pool.Close()

	b, _ := json.Marshal(map[string]any{"title": "Student Kit"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("student create: expected 403, got %d %s", rr.Code, rr.Body.String())
	}
}
