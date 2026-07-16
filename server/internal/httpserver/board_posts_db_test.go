package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func createBoardViaAPI(t *testing.T, h http.Handler, tok, cc string) string {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"title": "Post Board", "description": ""})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create board: %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("missing board id")
	}
	return id
}

func enrollSecondUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, cc, role string) (uuid.UUID, string) {
	t.Helper()
	em := fmt.Sprintf("board-post-%s-%d@test.com", role, time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	if _, err := pool.Exec(ctx,
		`INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, $3)`,
		courseID, uid, role,
	); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, uid, courseID, cc); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("grants: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, _ := signer.Sign(ctx, row.ID, em, "", "", nil)
	return uid, tok
}

func TestBoardPosts_TextCRUD_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, tok, cc)

	createBody, _ := json.Marshal(map[string]any{
		"contentType": "text",
		"title":       "Sticky",
		"body":        map[string]string{"html": "<p>Hello <strong>world</strong></p><script>x</script>"},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	postID, _ := created["id"].(string)
	if postID == "" {
		t.Fatal("missing post id")
	}
	bodyObj, _ := created["body"].(map[string]any)
	html, _ := bodyObj["html"].(string)
	if html == "" || contains(html, "script") {
		t.Fatalf("expected sanitized html, got %#v", created["body"])
	}

	rrList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	reqList.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rrList.Code, rrList.Body.String())
	}
	var list map[string]any
	_ = json.Unmarshal(rrList.Body.Bytes(), &list)
	posts, _ := list["posts"].([]any)
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	patchBody, _ := json.Marshal(map[string]any{"title": "Renamed"})
	rrPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID, bytes.NewReader(patchBody))
	reqPatch.Header.Set("Authorization", "Bearer "+tok)
	reqPatch.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rrPatch, reqPatch)
	if rrPatch.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rrPatch.Code, rrPatch.Body.String())
	}

	rrDel := httptest.NewRecorder()
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rrDel.Code, rrDel.Body.String())
	}
}

func TestBoardPosts_LinkRequiresURL_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, tok, cc)

	b, _ := json.Marshal(map[string]any{"contentType": "link", "title": "No URL"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestBoardPosts_StudentCannotEditOthers_InstructorCan_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, teacherTok, cc)

	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	_, student2Tok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")

	createBody, _ := json.Marshal(map[string]any{
		"contentType": "text",
		"title":       "Student note",
		"body":        map[string]string{"text": "mine"},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("student create: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	postID, _ := created["id"].(string)

	patchBody, _ := json.Marshal(map[string]any{"title": "Hijack"})
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID, bytes.NewReader(patchBody))
	req2.Header.Set("Authorization", "Bearer "+student2Tok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("other student edit: expected 403, got %d %s", rr2.Code, rr2.Body.String())
	}

	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID, bytes.NewReader(patchBody))
	req3.Header.Set("Authorization", "Bearer "+teacherTok)
	req3.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("teacher edit: expected 200, got %d %s", rr3.Code, rr3.Body.String())
	}
}

func TestBoardPosts_DrawingRoundTrip_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "student", true, true)
	defer pool.Close()

	// Students cannot create boards; insert board directly.
	var boardID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO board.boards (course_id, title, description, slug)
		SELECT id, 'Draw Board', '', 'draw-board'
		FROM course.courses WHERE course_code = $1
		RETURNING id
	`, cc).Scan(&boardID); err != nil {
		t.Fatalf("insert board: %v", err)
	}

	drawing := []map[string]any{
		{"type": "rect", "color": "#000", "width": 2, "x": 1, "y": 2, "w": 10, "h": 10},
	}
	createBody, _ := json.Marshal(map[string]any{
		"contentType": "drawing",
		"title":       "Sketch",
		"drawingData": drawing,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID.String()+"/posts", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create drawing: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	postID, _ := created["id"].(string)

	rrGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID.String()+"/posts/"+postID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rrGet.Code, rrGet.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rrGet.Body.Bytes(), &got)
	if got["drawingData"] == nil {
		t.Fatal("drawingData missing on reload")
	}
}

func TestBoardPosts_LinkPreviewSSRF_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, tok, cc)

	b, _ := json.Marshal(map[string]any{"url": "http://127.0.0.1/secret"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/link-preview", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for SSRF, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestBoardPosts_BlockedAttachmentNoURL_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	// Rebuild handler with AV scanning on.
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h = NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFVisualBoards: true, AvScanningEnabled: true}})

	boardID := createBoardViaAPI(t, h, tok, cc)
	var attID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO board.post_attachments (board_id, storage_key, file_name, mime_type, size_bytes, alt_text, scan_status)
		VALUES ($1::uuid, 'boards/x/blocked.png', 'blocked.png', 'image/png', 12, 'alt', 'blocked')
		RETURNING id
	`, boardID).Scan(&attID); err != nil {
		t.Fatalf("insert att: %v", err)
	}

	createBody, _ := json.Marshal(map[string]any{
		"contentType":  "image",
		"title":        "Blocked img",
		"attachmentId": attID.String(),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create image post: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	att, _ := created["attachment"].(map[string]any)
	if att == nil {
		t.Fatal("missing attachment")
	}
	if att["scanStatus"] != "blocked" {
		t.Fatalf("scanStatus=%v", att["scanStatus"])
	}
	if att["url"] != nil {
		t.Fatalf("blocked attachment must not expose url, got %v", att["url"])
	}
}

func contains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
