package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBoardModeration_ApprovalQueue_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	boardID := createBoardViaAPI(t, h, teacherTok, cc)

	// Enable approval mode.
	patchBoardJSON(t, h, teacherTok, cc, boardID, map[string]any{"moderationMode": "approval"})

	// Student posts — pending.
	postID := createTextPost(t, h, studentTok, cc, boardID, "Pending card", "hello class")
	studentPosts := listBoardPostsAPI(t, h, studentTok, cc, boardID)
	if len(studentPosts) != 1 {
		t.Fatalf("author should see own pending post, got %d", len(studentPosts))
	}
	if studentPosts[0]["status"] != "pending" {
		t.Fatalf("status=%v want pending", studentPosts[0]["status"])
	}

	// Peer student should not see it.
	_, peerTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	peerPosts := listBoardPostsAPI(t, h, peerTok, cc, boardID)
	if len(peerPosts) != 0 {
		t.Fatalf("peer should not see pending, got %d", len(peerPosts))
	}

	// Teacher queue + approve.
	queue := getModerationQueue(t, h, teacherTok, cc, boardID)
	pending, _ := queue["pending"].([]any)
	if len(pending) != 1 {
		t.Fatalf("queue pending=%d", len(pending))
	}
	approvePost(t, h, teacherTok, cc, boardID, postID)

	peerPosts = listBoardPostsAPI(t, h, peerTok, cc, boardID)
	if len(peerPosts) != 1 {
		t.Fatalf("peer should see approved, got %d", len(peerPosts))
	}
}

func TestBoardModeration_FilterBlockAndFlag_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	boardID := createBoardViaAPI(t, h, teacherTok, cc)

	patchBoardJSON(t, h, teacherTok, cc, boardID, map[string]any{"filterAction": "block"})
	rr := postText(t, h, studentTok, cc, boardID, "Bad", "what the fuck")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("block want 400, got %d %s", rr.Code, rr.Body.String())
	}
	if len(listBoardPostsAPI(t, h, teacherTok, cc, boardID)) != 0 {
		t.Fatal("blocked post must not be created")
	}

	patchBoardJSON(t, h, teacherTok, cc, boardID, map[string]any{"filterAction": "flag"})
	_ = createTextPost(t, h, studentTok, cc, boardID, "Flagged", "what the fuck")
	queue := getModerationQueue(t, h, teacherTok, cc, boardID)
	flagged, _ := queue["flagged"].([]any)
	if len(flagged) < 1 {
		t.Fatalf("expected filter flag in queue, got %#v", queue["flagged"])
	}
}

func TestBoardModeration_LockFreezeReport_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	boardID := createBoardViaAPI(t, h, teacherTok, cc)

	postID := createTextPost(t, h, studentTok, cc, boardID, "Report me", "normal text")

	// Report.
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"postId": postID, "reason": "hurtful"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/reports", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("report: %d %s", rr.Code, rr.Body.String())
	}
	var report map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &report)
	reportID, _ := report["id"].(string)

	queue := getModerationQueue(t, h, teacherTok, cc, boardID)
	reports, _ := queue["reports"].([]any)
	if len(reports) < 1 {
		t.Fatal("report missing from queue")
	}

	// Resolve with hide.
	rr2 := httptest.NewRecorder()
	body2, _ := json.Marshal(map[string]any{"action": "hide", "reason": "confirmed"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/reports/"+reportID+"/resolve", bytes.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+teacherTok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("resolve: %d %s", rr2.Code, rr2.Body.String())
	}
	if len(listBoardPostsAPI(t, h, studentTok, cc, boardID)) != 0 {
		t.Fatal("hidden post should not appear for student")
	}

	// Lock board — student cannot post.
	patchBoardJSON(t, h, teacherTok, cc, boardID, map[string]any{"locked": true})
	rr3 := postText(t, h, studentTok, cc, boardID, "Nope", "blocked by lock")
	if rr3.Code != http.StatusForbidden {
		t.Fatalf("locked post want 403, got %d %s", rr3.Code, rr3.Body.String())
	}

	// Freeze — student cannot post; teacher can.
	patchBoardJSON(t, h, teacherTok, cc, boardID, map[string]any{"locked": false, "freezeMinutes": 5})
	rr4 := postText(t, h, studentTok, cc, boardID, "Frozen", "should fail")
	if rr4.Code != http.StatusForbidden {
		t.Fatalf("frozen post want 403, got %d %s", rr4.Code, rr4.Body.String())
	}
	_ = createTextPost(t, h, teacherTok, cc, boardID, "Teacher ok", "managers can post while frozen")

	// Expired freeze resumes posting.
	past := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	patchBoardJSON(t, h, teacherTok, cc, boardID, map[string]any{"frozenUntil": past})
	_ = createTextPost(t, h, studentTok, cc, boardID, "Thawed", "ok again")

	// Audit log has entries.
	rr5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/moderation/log", nil)
	req5.Header.Set("Authorization", "Bearer "+teacherTok)
	h.ServeHTTP(rr5, req5)
	if rr5.Code != http.StatusOK {
		t.Fatalf("log: %d %s", rr5.Code, rr5.Body.String())
	}
	var logBody map[string]any
	_ = json.Unmarshal(rr5.Body.Bytes(), &logBody)
	entries, _ := logBody["entries"].([]any)
	if len(entries) < 1 {
		t.Fatal("expected moderation log entries")
	}
}

func patchBoardJSON(t *testing.T, h http.Handler, tok, cc, boardID string, patch map[string]any) {
	t.Helper()
	b, _ := json.Marshal(patch)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch board: %d %s", rr.Code, rr.Body.String())
	}
}

func createTextPost(t *testing.T, h http.Handler, tok, cc, boardID, title, text string) string {
	t.Helper()
	rr := postText(t, h, tok, cc, boardID, title, text)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("missing post id")
	}
	return id
}

func postText(t *testing.T, h http.Handler, tok, cc, boardID, title, text string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(map[string]any{
		"contentType": "text",
		"title":       title,
		"body":        map[string]string{"text": text},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	return rr
}

func listBoardPostsAPI(t *testing.T, h http.Handler, tok, cc, boardID string) []map[string]any {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list posts: %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	raw, _ := body["posts"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func getModerationQueue(t *testing.T, h http.Handler, tok, cc, boardID string) map[string]any {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/moderation/queue", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("queue: %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	return body
}

func approvePost(t *testing.T, h http.Handler, tok, cc, boardID, postID string) {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/approve", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", rr.Code, rr.Body.String())
	}
}
