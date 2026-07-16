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

func setupBoardTest(t *testing.T, ctx context.Context, role string, courseFlag, masterFlag bool) (
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

	em := fmt.Sprintf("boards-%s-%d@test.com", role, time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		pool.Close()
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	// Format: C-[A-Z0-9]{6} (courses_course_code_format).
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])

	var courseID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Boards Test', $2) RETURNING id`,
		cc, uid,
	).Scan(&courseID); err != nil {
		pool.Close()
		t.Fatalf("course: %v", err)
	}
	if courseFlag {
		if _, err := pool.Exec(ctx, `UPDATE course.courses SET visual_boards_enabled = true WHERE id = $1`, courseID); err != nil {
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
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFVisualBoards: masterFlag}})
	return pool, h, tok, cc, courseID
}

func TestBoards_FeatureGate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", false, true)
	defer pool.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("gate: expected 404 when course flag off, got %d %s", rr.Code, rr.Body.String())
	}

	b, _ := json.Marshal(map[string]any{
		"notebookEnabled":     true,
		"feedEnabled":         false,
		"calendarEnabled":     true,
		"questionBankEnabled": false,
		"lockdownModeEnabled": false,
		"discussionsEnabled":  false,
		"visualBoardsEnabled": true,
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
	if body["visualBoardsEnabled"] != true {
		t.Fatalf("expected visualBoardsEnabled=true, got %v", body["visualBoardsEnabled"])
	}

	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("list after enable: %d %s", rr3.Code, rr3.Body.String())
	}
}

func TestBoards_MasterFlagOff_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, false)
	defer pool.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("master off: expected 404, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestBoards_FullCRUD_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	var boardID string
	{
		b, _ := json.Marshal(map[string]any{"title": "Brainstorm Wall", "description": "Week 1 ideas"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		boardID, _ = body["id"].(string)
		if boardID == "" {
			t.Fatal("create: missing id")
		}
		if body["title"] != "Brainstorm Wall" {
			t.Fatalf("create: title %v", body["title"])
		}
		if body["slug"] == "" {
			t.Fatal("create: missing slug")
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		boards, _ := body["boards"].([]any)
		if len(boards) != 1 {
			t.Fatalf("list: expected 1, got %d", len(boards))
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: %d %s", rr.Code, rr.Body.String())
		}
	}

	{
		b, _ := json.Marshal(map[string]any{"title": "Renamed Wall"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("rename: %d %s", rr.Code, rr.Body.String())
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+cc+"/boards/"+boardID, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("archive: %d %s", rr.Code, rr.Body.String())
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		boards, _ := body["boards"].([]any)
		if len(boards) != 0 {
			t.Fatalf("list after archive: expected 0, got %d", len(boards))
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards?includeArchived=true", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, req)
		var body map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		boards, _ := body["boards"].([]any)
		if len(boards) != 1 {
			t.Fatalf("includeArchived: expected 1, got %d", len(boards))
		}
	}

	if _, err := pool.Exec(ctx, `DELETE FROM course.courses WHERE id = $1`, courseID); err != nil {
		t.Fatalf("delete course: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM board.boards WHERE id = $1`, boardID).Scan(&n); err != nil {
		t.Fatalf("count boards: %v", err)
	}
	if n != 0 {
		t.Fatalf("cascade: expected 0 boards, got %d", n)
	}
}

func TestBoards_StudentCannotCreate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "student", true, true)
	defer pool.Close()

	b, _ := json.Marshal(map[string]any{"title": "Nope"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("student create: expected 403, got %d %s", rr.Code, rr.Body.String())
	}
}
