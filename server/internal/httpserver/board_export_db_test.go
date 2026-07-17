package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestBoardExport_CSV_PDF_Image_QR_Embed_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	tmp := t.TempDir()
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			FFVisualBoards:  true,
			CourseFilesRoot: tmp,
			PublicWebOrigin: "http://localhost:5173",
		},
	})

	authHdr := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", strings.NewReader(`{"title":"Export Board"}`))
	authHdr(req)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create board: %d %s", rec.Code, rec.Body.String())
	}
	var boardResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &boardResp)
	boardID, _ := boardResp["id"].(string)

	secRec := httptest.NewRecorder()
	secReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/sections", strings.NewReader(`{"title":"Ideas"}`))
	authHdr(secReq)
	h.ServeHTTP(secRec, secReq)
	var sec map[string]any
	_ = json.Unmarshal(secRec.Body.Bytes(), &sec)
	secID, _ := sec["id"].(string)

	postRec := httptest.NewRecorder()
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts",
		strings.NewReader(`{"contentType":"text","title":"Visible","body":{"text":"hello"} }`))
	authHdr(postReq)
	h.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", postRec.Code, postRec.Body.String())
	}
	var createdPost map[string]any
	_ = json.Unmarshal(postRec.Body.Bytes(), &createdPost)
	postID, _ := createdPost["id"].(string)
	if postID != "" && secID != "" {
		arrRec := httptest.NewRecorder()
		arrReq := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+postID+"/arrange",
			strings.NewReader(`{"sectionId":"`+secID+`"}`))
		authHdr(arrReq)
		h.ServeHTTP(arrRec, arrReq)
		if arrRec.Code != http.StatusOK {
			t.Fatalf("arrange post: %d %s", arrRec.Code, arrRec.Body.String())
		}
	}

	hideRec := httptest.NewRecorder()
	hideReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts",
		strings.NewReader(`{"contentType":"text","title":"Hidden","body":{"text":"secret"} }`))
	authHdr(hideReq)
	h.ServeHTTP(hideRec, hideReq)
	var hidePost map[string]any
	_ = json.Unmarshal(hideRec.Body.Bytes(), &hidePost)
	if hideID, _ := hidePost["id"].(string); hideID != "" {
		hr := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts/"+hideID+"/hide", strings.NewReader(`{"reason":"test"}`))
		authHdr(hr)
		h.ServeHTTP(httptest.NewRecorder(), hr)
	}

	for _, format := range []string{"csv", "pdf", "image"} {
		expRec := httptest.NewRecorder()
		expReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/export",
			strings.NewReader(`{"format":"`+format+`"}`))
		authHdr(expReq)
		h.ServeHTTP(expRec, expReq)
		if expRec.Code != http.StatusAccepted {
			t.Fatalf("export %s: %d %s", format, expRec.Code, expRec.Body.String())
		}
		var expResp struct {
			Job struct {
				Status      string  `json:"status"`
				DownloadURL *string `json:"downloadUrl"`
			} `json:"job"`
		}
		_ = json.Unmarshal(expRec.Body.Bytes(), &expResp)
		if expResp.Job.Status != "done" || expResp.Job.DownloadURL == nil {
			t.Fatalf("export %s not done: %+v body=%s", format, expResp.Job, expRec.Body.String())
		}
		dlRec := httptest.NewRecorder()
		dlReq := httptest.NewRequest(http.MethodGet, *expResp.Job.DownloadURL, nil)
		dlReq.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(dlRec, dlReq)
		if dlRec.Code != http.StatusOK {
			t.Fatalf("download %s: %d %s", format, dlRec.Code, dlRec.Body.String())
		}
		body := dlRec.Body.Bytes()
		switch format {
		case "csv":
			s := string(body)
			if !strings.Contains(s, "Visible") || !strings.Contains(s, "Ideas") {
				t.Fatalf("csv missing content: %s", s)
			}
			if strings.Contains(s, "Hidden") || strings.Contains(s, "secret") {
				t.Fatalf("csv leaked hidden card: %s", s)
			}
		case "pdf":
			if !bytes.HasPrefix(body, []byte("%PDF")) {
				t.Fatalf("pdf magic: %q", body[:min(8, len(body))])
			}
		case "image":
			if !bytes.HasPrefix(body, []byte("\x89PNG")) {
				t.Fatalf("png magic: %q", body[:min(8, len(body))])
			}
		}
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+cc+"/boards/"+boardID,
		strings.NewReader(`{"attribution":"anonymous"}`))
	authHdr(patchReq)
	h.ServeHTTP(httptest.NewRecorder(), patchReq)

	expRec := httptest.NewRecorder()
	expReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/export",
		strings.NewReader(`{"format":"csv"}`))
	authHdr(expReq)
	h.ServeHTTP(expRec, expReq)
	var expResp struct {
		Job struct {
			DownloadURL *string `json:"downloadUrl"`
		} `json:"job"`
	}
	_ = json.Unmarshal(expRec.Body.Bytes(), &expResp)
	if expResp.Job.DownloadURL == nil {
		t.Fatalf("anon csv: %s", expRec.Body.String())
	}
	dlRec := httptest.NewRecorder()
	dlReq := httptest.NewRequest(http.MethodGet, *expResp.Job.DownloadURL, nil)
	dlReq.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(dlRec, dlReq)
	if strings.Contains(dlRec.Body.String(), "user-") {
		t.Fatalf("anonymous CSV leaked author: %s", dlRec.Body.String())
	}

	qrRec := httptest.NewRecorder()
	qrReq := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/qr?format=png", nil)
	qrReq.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(qrRec, qrReq)
	if qrRec.Code != http.StatusOK {
		t.Fatalf("qr: %d %s", qrRec.Code, qrRec.Body.String())
	}
	wantURL := "http://localhost:5173/courses/" + cc + "/boards/" + boardID
	if got := qrRec.Header().Get("X-Board-Access-Url"); got != wantURL {
		t.Fatalf("qr url: got %q want %q", got, wantURL)
	}
	if !bytes.HasPrefix(qrRec.Body.Bytes(), []byte("\x89PNG")) {
		t.Fatal("qr not png")
	}

	embRec := httptest.NewRecorder()
	embReq := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/embed", nil)
	embReq.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(embRec, embReq)
	if embRec.Code != http.StatusOK {
		t.Fatalf("embed: %d %s", embRec.Code, embRec.Body.String())
	}
	var emb map[string]any
	_ = json.Unmarshal(embRec.Body.Bytes(), &emb)
	mode, _ := emb["mode"].(string)
	if mode != "interactive" && mode != "readonly" {
		t.Fatalf("embed mode: %v", emb["mode"])
	}

	var found bool
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if strings.HasSuffix(path, ".csv") || strings.HasSuffix(path, ".pdf") || strings.HasSuffix(path, ".png") {
				found = true
			}
		}
		return nil
	})
	if !found {
		t.Fatal("expected export files under CourseFilesRoot")
	}
}

func TestBoardExport_ForbiddenWithoutManage_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, teacherTok, cc, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", strings.NewReader(`{"title":"B"}`))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	var boardResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &boardResp)
	boardID, _ := boardResp["id"].(string)

	// Enroll a student in the same course without item:create.
	em := "boards-student-export-" + time.Now().Format("150405.000000000") + "@test.com"
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')`,
		courseID, row.ID,
	); err != nil {
		t.Fatal(err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	studentTok, _ := signer.Sign(ctx, row.ID, em, "", "", nil)

	expRec := httptest.NewRecorder()
	expReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/export",
		strings.NewReader(`{"format":"csv"}`))
	expReq.Header.Set("Authorization", "Bearer "+studentTok)
	expReq.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(expRec, expReq)
	if expRec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d %s", expRec.Code, expRec.Body.String())
	}
}
