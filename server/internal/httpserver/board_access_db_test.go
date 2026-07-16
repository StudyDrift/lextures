package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/board"
)

func TestBoardAccess_ContributorPolicyAndAnonAttribution_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, teacherTok, courseCode, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	studentID, studentTok := enrollSecondUser(t, ctx, pool, courseID, courseCode, "student")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/boards", bytes.NewBufferString(`{"title":"Access board"}`))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create board: %d %s", rr.Code, rr.Body.String())
	}
	var created board.Board
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	patch := map[string]any{
		"canPost": true, "canInteract": true, "canArrange": false,
		"attribution": "anon_to_peers",
	}
	body, _ := json.Marshal(patch)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/boards/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/boards/"+created.ID+"/posts",
		bytes.NewBufferString(`{"contentType":"text","body":{"text":"hello","html":"hello"}}`))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", rr.Code, rr.Body.String())
	}
	var post map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &post)
	postID, _ := post["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/boards/"+created.ID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list posts student: %d", rr.Code)
	}
	var list struct {
		Posts []map[string]any `json:"posts"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if len(list.Posts) == 0 {
		t.Fatal("expected posts")
	}
	if list.Posts[0]["authorId"] != nil {
		t.Fatalf("student should not see authorId, got %v", list.Posts[0]["authorId"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/boards/"+created.ID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if list.Posts[0]["authorId"] != studentID.String() {
		t.Fatalf("teacher should see authorId %s, got %v", studentID, list.Posts[0]["authorId"])
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/boards/"+created.ID+"/posts/"+postID+"/arrange",
		bytes.NewBufferString(`{"sortIndex":1}`))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected arrange 403, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestBoardAccess_ShareLinkPasswordRevoke_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, _, teacherTok, courseCode, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{
		FFVisualBoards: true, FFBoardsExternalSharing: true,
	}})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/boards", bytes.NewBufferString(`{"title":"Link board"}`))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created board.Board
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/boards/"+created.ID+"/shares",
		bytes.NewBufferString(`{"capability":"contribute","password":"s3cret-pass"}`))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create share: %d %s", rr.Code, rr.Body.String())
	}
	var share map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &share)
	token, _ := share["token"].(string)
	shareID, _ := share["id"].(string)
	if token == "" {
		t.Fatal("expected token")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/board-links/"+token, nil)
	req.Header.Set("X-Board-Share-Password", "wrong")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/board-links/"+token, nil)
	req.Header.Set("X-Board-Share-Password", "s3cret-pass")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("resolve: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/board-links/"+token+"/posts",
		bytes.NewBufferString(`{"displayName":"Guest","contentType":"text","body":{"text":"hi","html":"hi"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Board-Share-Password", "s3cret-pass")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("guest post: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+courseCode+"/boards/"+created.ID+"/shares/"+shareID, nil)
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d", rr.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/board-links/"+token, nil)
	req.Header.Set("X-Board-Share-Password", "s3cret-pass")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("revoked link should 404, got %d", rr.Code)
	}
}

func TestBoardAccess_ExternalSharingDisabled_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, teacherTok, courseCode, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/boards", bytes.NewBufferString(`{"title":"No ext"}`))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var created board.Board
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/boards/"+created.ID,
		bytes.NewBufferString(`{"visibility":"public"}`))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when external sharing off, got %d %s", rr.Code, rr.Body.String())
	}
}
