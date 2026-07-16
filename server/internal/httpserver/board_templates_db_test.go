package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/board"
)

func TestBoardTemplates_ListBuiltinsAndInstantiate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/board-templates?scope=builtin&locale=en", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list builtins: %d %s", rr.Code, rr.Body.String())
	}
	var listBody struct {
		Templates []map[string]any `json:"templates"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &listBody)
	if len(listBody.Templates) < 8 {
		t.Fatalf("expected >=8 builtins, got %d", len(listBody.Templates))
	}

	// AC-1: Exit ticket template
	body, _ := json.Marshal(map[string]any{"title": "", "description": ""})
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/courses/"+cc+"/boards?from=template:"+board.BuiltinExitTicketID,
		bytes.NewReader(body),
	)
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept-Language", "en")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("instantiate exit ticket: %d %s", rr2.Code, rr2.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &created)
	boardID, _ := created["id"].(string)
	if boardID == "" {
		t.Fatal("missing board id")
	}
	if created["layout"] != "stream" {
		t.Fatalf("layout: %v", created["layout"])
	}
	if created["title"] != "Exit ticket" {
		t.Fatalf("title: %v", created["title"])
	}

	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("list posts: %d %s", rr3.Code, rr3.Body.String())
	}
	var postsBody struct {
		Posts []map[string]any `json:"posts"`
	}
	_ = json.Unmarshal(rr3.Body.Bytes(), &postsBody)
	if len(postsBody.Posts) != 1 {
		t.Fatalf("expected 1 seed post, got %d", len(postsBody.Posts))
	}
	if postsBody.Posts[0]["authorId"] == nil || postsBody.Posts[0]["authorId"] == "" {
		t.Fatal("seed post must be attributed to instructor")
	}
}

func TestBoardTemplates_StructureOnlyDuplicate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	srcID := createBoardViaAPI(t, h, tok, cc)
	// Add a section + student-looking post
	secBody, _ := json.Marshal(map[string]any{"title": "Ideas"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+srcID+"/sections", bytes.NewReader(secBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("section: %d %s", rr.Code, rr.Body.String())
	}

	postBody, _ := json.Marshal(map[string]any{
		"contentType": "text",
		"title":       "Student idea",
		"body":        map[string]string{"text": "hello"},
	})
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+srcID+"/posts", bytes.NewReader(postBody))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("post: %d %s", rr2.Code, rr2.Body.String())
	}

	dupBody, _ := json.Marshal(map[string]any{"title": "Copy structure"})
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/courses/"+cc+"/boards?from=board:"+srcID+"&mode=structure",
		bytes.NewReader(dupBody),
	)
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusCreated {
		t.Fatalf("dup: %d %s", rr3.Code, rr3.Body.String())
	}
	var copied map[string]any
	_ = json.Unmarshal(rr3.Body.Bytes(), &copied)
	copyID, _ := copied["id"].(string)

	rr4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+copyID+"/posts", nil)
	req4.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr4, req4)
	var postsBody struct {
		Posts []map[string]any `json:"posts"`
	}
	_ = json.Unmarshal(rr4.Body.Bytes(), &postsBody)
	if len(postsBody.Posts) != 0 {
		t.Fatalf("structure-only must have zero posts, got %d", len(postsBody.Posts))
	}

	rr5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+copyID+"/sections", nil)
	req5.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr5, req5)
	var secsBody struct {
		Sections []map[string]any `json:"sections"`
	}
	_ = json.Unmarshal(rr5.Body.Bytes(), &secsBody)
	if len(secsBody.Sections) != 1 {
		t.Fatalf("expected section copied, got %d", len(secsBody.Sections))
	}
}

func TestBoardTemplates_FullCopyIndependentAttachments_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	root := t.TempDir()
	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	srcID := createBoardViaAPI(t, h, tok, cc)
	srcKey := fmt.Sprintf("boards/%s/attachments/%s/img.bin", cc, srcID)
	srcPath := filepath.Join(root, filepath.FromSlash(srcKey))
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte("image-bytes-v1")
	if err := os.WriteFile(srcPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	var teacherID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT created_by FROM board.boards WHERE id = $1`, srcID).Scan(&teacherID); err != nil {
		t.Fatal(err)
	}
	att, err := board.CreateAttachment(ctx, pool, cc, srcID, teacherID, srcKey, "shot.png", "image/png", "alt", board.ScanClean, int64(len(payload)))
	if err != nil || att == nil {
		t.Fatalf("att: %v", err)
	}
	attID := att.ID
	_, err = board.CreatePost(ctx, pool, cc, srcID, teacherID, board.CreatePostInput{
		ContentType:  board.ContentTypeImage,
		Title:        "Photo",
		AttachmentID: &attID,
	}, nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	// Handler from setupBoardTest does not have CourseFilesRoot; call repo CopyBoard with local copier.
	copier := &boardBlobCopier{root: root}
	copied, err := board.CopyBoard(ctx, pool, cc, srcID, cc, teacherID, board.CopyBoardOpts{
		Mode:       board.CopyModeFull,
		Title:      "Full copy",
		AuthorID:   teacherID,
		BlobCopier: copier,
	})
	if err != nil || copied == nil {
		t.Fatalf("copy: %v", err)
	}
	posts, err := board.ListPosts(ctx, pool, cc, copied.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].Attachment == nil {
		t.Fatalf("expected 1 image post with attachment, got %+v", posts)
	}
	if posts[0].Attachment.StorageKey == srcKey {
		t.Fatal("attachment storage key must not be shared")
	}
	destPath := filepath.Join(root, filepath.FromSlash(posts[0].Attachment.StorageKey))
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("dest bytes mismatch")
	}
	// Independence: delete source file; dest remains.
	_ = os.Remove(srcPath)
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("dest should remain after source delete: %v", err)
	}
}

func TestBoardTemplates_SaveAsTemplateCourse_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	boardID := createBoardViaAPI(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{
		"scope":        "course",
		"title":        "My course template",
		"description":  "reuse",
		"includePosts": false,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/save-as-template", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("save: %d %s", rr.Code, rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/board-templates?scope=course&courseCode="+cc, nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rr2.Code, rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), "My course template") {
		t.Fatalf("expected saved template in gallery: %s", rr2.Body.String())
	}
}

func TestBoardTemplates_CrossCourseDeniedWithoutCreate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, srcCC, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	srcBoard := createBoardViaAPI(t, h, teacherTok, srcCC)

	// Student in another course (no item:create).
	pool2, h2, studentTok, targetCC, _ := setupBoardTest(t, ctx, "student", true, true)
	defer pool2.Close()
	_ = h2

	// Student cannot create in target; also can't see source. Use teacher on target without create — use student.
	body, _ := json.Marshal(map[string]any{"title": "X"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/courses/"+targetCC+"/boards?from=board:"+srcBoard+"&mode=structure",
		bytes.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	h2.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Fatalf("expected forbid/not found for cross-course without create, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestBoardTemplates_LocaleOnInstantiate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	body, _ := json.Marshal(map[string]any{"title": "", "description": ""})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/courses/"+cc+"/boards?from=template:"+board.BuiltinExitTicketID+"&locale=es",
		bytes.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created["title"] != "Ticket de salida" {
		t.Fatalf("expected localized title, got %v", created["title"])
	}
}
