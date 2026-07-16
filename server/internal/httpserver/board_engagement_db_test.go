package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func createTextPostViaAPI(t *testing.T, h http.Handler, tok, cc, boardID, title string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"contentType": "text",
		"title":       title,
		"body":        map[string]string{"text": title, "html": "<p>" + title + "</p>"},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing post id")
	}
	return id
}

func patchBoardReactionMode(t *testing.T, h http.Handler, tok, cc, boardID, mode string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"reactionMode": mode})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch reactionMode: %d %s", rr.Code, rr.Body.String())
	}
}

func TestBoardReactions_LikeToggleIdempotent_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, tok, cc)
	patchBoardReactionMode(t, h, tok, cc, boardID, "like")
	postID := createTextPostViaAPI(t, h, tok, cc, boardID, "Idea")

	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")

	put := func(expectActive bool, expectCount float64) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"kind": "like"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/reaction", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+studentTok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("put reaction: %d %s", rr.Code, rr.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		if out["active"] != expectActive {
			t.Fatalf("active=%v want %v (%s)", out["active"], expectActive, rr.Body.String())
		}
		if out["reactionCount"] != expectCount {
			t.Fatalf("reactionCount=%v want %v", out["reactionCount"], expectCount)
		}
	}

	put(true, 1)
	put(false, 0)
	put(true, 1)

	// Post list includes aggregates.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list posts: %d %s", rr.Code, rr.Body.String())
	}
	var list map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	posts, _ := list["posts"].([]any)
	if len(posts) != 1 {
		t.Fatalf("posts len=%d", len(posts))
	}
	row, _ := posts[0].(map[string]any)
	if row["reactionCount"] != float64(1) {
		t.Fatalf("list reactionCount=%v", row["reactionCount"])
	}
	if row["myReaction"] == nil {
		t.Fatal("expected myReaction")
	}
	if row["commentCount"] != float64(0) {
		t.Fatalf("commentCount=%v", row["commentCount"])
	}
}

func TestBoardReactions_StarAverage_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, tok, cc)
	patchBoardReactionMode(t, h, tok, cc, boardID, "star")
	postID := createTextPostViaAPI(t, h, tok, cc, boardID, "Rated")

	_, s1 := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	_, s2 := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	_, s3 := enrollSecondUser(t, ctx, pool, courseID, cc, "student")

	rate := func(tok string, stars float64) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"kind": "star", "value": stars})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/reaction", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("star: %d %s", rr.Code, rr.Body.String())
		}
	}
	rate(s1, 4)
	rate(s2, 5)
	rate(s3, 3)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	var list map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	posts, _ := list["posts"].([]any)
	row, _ := posts[0].(map[string]any)
	if row["avgStars"] != 4.0 {
		t.Fatalf("avgStars=%v want 4.0", row["avgStars"])
	}
	if row["reactionCount"] != float64(3) {
		t.Fatalf("reactionCount=%v", row["reactionCount"])
	}
}

func TestBoardReactions_GradeVisibility_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, teacherTok, cc)
	patchBoardReactionMode(t, h, teacherTok, cc, boardID, "grade")

	authorID, authorTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	_, peerTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")

	// Author creates the card.
	postID := createTextPostViaAPI(t, h, authorTok, cc, boardID, "Submission")

	// Peer cannot grade.
	body, _ := json.Marshal(map[string]any{"kind": "grade", "value": 88})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/reaction", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+peerTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("peer grade: expected 403, got %d %s", rr.Code, rr.Body.String())
	}

	// Teacher grades.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/reaction", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("teacher grade: %d %s", rr.Code, rr.Body.String())
	}

	// Author sees grade.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+authorTok)
	h.ServeHTTP(rr, req)
	var list map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	row, _ := list["posts"].([]any)[0].(map[string]any)
	if row["grade"] != float64(88) {
		t.Fatalf("author grade=%v", row["grade"])
	}

	// Peer does not see grade.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+peerTok)
	h.ServeHTTP(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	row, _ = list["posts"].([]any)[0].(map[string]any)
	if _, ok := row["grade"]; ok {
		t.Fatalf("peer must not see grade: %v", row)
	}
	_ = authorID
}

func TestBoardComments_ThreadAndHide_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	boardID := createBoardViaAPI(t, h, teacherTok, cc)
	postID := createTextPostViaAPI(t, h, teacherTok, cc, boardID, "Discuss")
	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")

	// Root comment.
	body, _ := json.Marshal(map[string]any{
		"body": map[string]string{"text": "Nice idea", "html": "<p>Nice idea</p><script>x</script>"},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/comments", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create comment: %d %s", rr.Code, rr.Body.String())
	}
	var root map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &root)
	rootID, _ := root["id"].(string)
	bodyObj, _ := root["body"].(map[string]any)
	html, _ := bodyObj["html"].(string)
	if html == "" || containsScript(html) {
		t.Fatalf("expected sanitized html, got %q", html)
	}

	// Reply.
	replyBody, _ := json.Marshal(map[string]any{
		"body":     map[string]string{"text": "Thanks!", "html": "<p>Thanks!</p>"},
		"parentId": rootID,
	})
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/comments", bytes.NewReader(replyBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("reply: %d %s", rr.Code, rr.Body.String())
	}
	var reply map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &reply)
	if reply["parentId"] != rootID {
		t.Fatalf("parentId=%v", reply["parentId"])
	}

	// Comment count on post list.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	h.ServeHTTP(rr, req)
	var list map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	row, _ := list["posts"].([]any)[0].(map[string]any)
	if row["commentCount"] != float64(2) {
		t.Fatalf("commentCount=%v", row["commentCount"])
	}

	// Teacher hides root.
	hideBody, _ := json.Marshal(map[string]any{"hidden": true})
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/comments/"+rootID, bytes.NewReader(hideBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("hide: %d %s", rr.Code, rr.Body.String())
	}

	// Student list omits hidden (+ nested reply may still show depending on parent hide — both listed independently).
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/comments", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	h.ServeHTTP(rr, req)
	var studentComments map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &studentComments)
	sc, _ := studentComments["comments"].([]any)
	for _, c := range sc {
		m, _ := c.(map[string]any)
		if m["id"] == rootID {
			t.Fatal("student should not see hidden root")
		}
	}

	// Teacher still sees hidden row for audit.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/comments", nil)
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	h.ServeHTTP(rr, req)
	var teacherComments map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &teacherComments)
	foundHidden := false
	for _, c := range teacherComments["comments"].([]any) {
		m, _ := c.(map[string]any)
		if m["id"] == rootID && m["hidden"] == true {
			foundHidden = true
		}
	}
	if !foundHidden {
		t.Fatal("teacher should see hidden comment for audit")
	}
}

func TestBoardGradeSync_WritesGradebook_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	// Create a structure item + module assignment for grade sync.
	var moduleID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO course.course_structure_items (course_id, kind, title, sort_order, published, archived)
		VALUES ($1, 'module', 'Board Module', 0, true, false)
		RETURNING id
	`, courseID).Scan(&moduleID); err != nil {
		t.Fatalf("create module: %v", err)
	}
	var itemID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO course.course_structure_items (course_id, parent_id, kind, title, sort_order, published, archived)
		VALUES ($1, $2, 'assignment', 'Board Rubric', 0, true, false)
		RETURNING id
	`, courseID, moduleID).Scan(&itemID); err != nil {
		t.Fatalf("create assignment item: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO course.module_assignments (structure_item_id, points_worth, posting_policy)
		VALUES ($1, 100, 'automatic')
		ON CONFLICT (structure_item_id) DO UPDATE SET posting_policy = 'automatic'
	`, itemID); err != nil {
		t.Fatalf("module_assignments: %v", err)
	}

	boardID := createBoardViaAPI(t, h, teacherTok, cc)
	patchBody, _ := json.Marshal(map[string]any{
		"reactionMode": "grade",
		"assignmentId": itemID.String(),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("link assignment: %d %s", rr.Code, rr.Body.String())
	}

	_, studentTok := enrollSecondUser(t, ctx, pool, courseID, cc, "student")
	postID := createTextPostViaAPI(t, h, studentTok, cc, boardID, "Graded work")

	gradeBody, _ := json.Marshal(map[string]any{"kind": "grade", "value": 91})
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/reaction", bytes.NewReader(gradeBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("grade: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/grade-sync", nil)
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("grade-sync: %d %s", rr.Code, rr.Body.String())
	}

	var pts float64
	err := pool.QueryRow(ctx, `
		SELECT points_earned FROM course.course_grades
		WHERE course_id = $1 AND module_item_id = $2
	`, courseID, itemID).Scan(&pts)
	if err != nil {
		t.Fatalf("read gradebook: %v", err)
	}
	if pts != 91 {
		t.Fatalf("points_earned=%v want 91", pts)
	}
}

func containsScript(s string) bool {
	return bytes.Contains([]byte(s), []byte("<script"))
}
