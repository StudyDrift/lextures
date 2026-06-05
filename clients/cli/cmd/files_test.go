package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newFilesServer builds a minimal test server covering course-files API endpoints.
func newFilesServer(t *testing.T, mux map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := mux[key]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	}))
}

func resetFilesFlags() {
	filesListFlags.course = ""
	filesListFlags.folder = ""
	filesMkdirFlags.course = ""
	filesMkdirFlags.name = ""
	filesMkdirFlags.parent = ""
	filesUploadFlags.course = ""
	filesUploadFlags.folder = ""
	filesUploadFlags.quiet = false
	filesDownloadFlags.course = ""
	filesDownloadFlags.out = ""
	filesRenameFlags.course = ""
	filesRenameFlags.item = ""
	filesRenameFlags.folder = ""
	filesMoveFlags.course = ""
	filesMoveFlags.item = ""
	filesMoveFlags.folder = ""
	filesMoveFlags.to = ""
	filesDeleteFlags.course = ""
	filesDeleteFlags.item = ""
	filesDeleteFlags.folder = ""
	filesDeleteFlags.force = false
	filesDeleteInput = nil
}

func sampleFolder(id, name string) fileFolder {
	return fileFolder{
		ID:        id,
		CourseID:  "course-1",
		Name:      name,
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}
}

func sampleFileItem(id, display string) fileItem {
	return fileItem{
		ID:               id,
		CourseID:         "course-1",
		StorageKey:       "key/" + id,
		OriginalFilename: display,
		DisplayName:      display,
		MimeType:         "application/pdf",
		ByteSize:         2048,
		CreatedAt:        "2026-01-01T00:00:00Z",
		UpdatedAt:        "2026-01-02T00:00:00Z",
	}
}

// ============================================================
// files list
// ============================================================

func TestFilesList_Root(t *testing.T) {
	contents := folderContents{
		Folders: []fileFolder{sampleFolder("f1", "Lectures")},
		Files:   []fileItem{sampleFileItem("i1", "syllabus.pdf")},
	}
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/courses/CS101/files": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(contents)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesListFlags.course = "CS101"

	var out bytes.Buffer
	filesListCmd.SetOut(&out)
	if err := filesListCmd.RunE(filesListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Lectures") {
		t.Errorf("output = %q; want folder name", output)
	}
	if !strings.Contains(output, "syllabus.pdf") {
		t.Errorf("output = %q; want file name", output)
	}
}

func TestFilesList_Folder(t *testing.T) {
	contents := folderContents{
		FolderID: strPtr("f1"),
		Folders:  []fileFolder{},
		Files:    []fileItem{sampleFileItem("i2", "lecture1.pdf")},
	}
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/courses/CS101/files/folders/f1": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(contents)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesListFlags.course = "CS101"
	filesListFlags.folder = "f1"

	var out bytes.Buffer
	filesListCmd.SetOut(&out)
	if err := filesListCmd.RunE(filesListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "lecture1.pdf") {
		t.Errorf("output = %q; want lecture1.pdf", out.String())
	}
}

func TestFilesList_JSON(t *testing.T) {
	contents := folderContents{
		Files: []fileItem{sampleFileItem("i1", "notes.txt")},
	}
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/courses/CS101/files": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(contents)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesListFlags.course = "CS101"

	prev := globalFlags.jsonOut
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = prev }()

	var out bytes.Buffer
	filesListCmd.SetOut(&out)
	if err := filesListCmd.RunE(filesListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var decoded folderContents
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded.Files) != 1 || decoded.Files[0].DisplayName != "notes.txt" {
		t.Errorf("decoded = %+v", decoded)
	}
}

// ============================================================
// files mkdir
// ============================================================

func TestFilesMkdir_Success(t *testing.T) {
	folder := sampleFolder("f99", "Homework")
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/courses/CS101/files/folders": func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["name"] != "Homework" {
				http.Error(w, "bad name", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(folder)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesMkdirFlags.course = "CS101"
	filesMkdirFlags.name = "Homework"

	var out bytes.Buffer
	filesMkdirCmd.SetOut(&out)
	if err := filesMkdirCmd.RunE(filesMkdirCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "f99") {
		t.Errorf("output = %q; want folder ID", out.String())
	}
}

// ============================================================
// files upload (local storage path)
// ============================================================

func TestFilesUpload_LocalStorage(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "notes.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	item := sampleFileItem("i42", "notes.txt")
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/courses/CS101/files/items": func(w http.ResponseWriter, r *http.Request) {
			// Local storage: server returns FileItem directly (has an id field).
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			raw, _ := json.Marshal(map[string]any{
				"id":               item.ID,
				"courseId":         item.CourseID,
				"objectKey":        item.StorageKey,
				"originalFilename": item.OriginalFilename,
				"displayName":      item.DisplayName,
				"mimeType":         item.MimeType,
				"byteSize":         item.ByteSize,
				"createdAt":        item.CreatedAt,
				"updatedAt":        item.UpdatedAt,
			})
			_, _ = w.Write(raw)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesUploadFlags.course = "CS101"
	filesUploadFlags.quiet = true

	var out bytes.Buffer
	filesUploadCmd.SetOut(&out)
	if err := filesUploadCmd.RunE(filesUploadCmd, []string{testFile}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "i42") {
		t.Errorf("output = %q; want item ID", out.String())
	}
}

func TestFilesUpload_PathTraversalRejected(t *testing.T) {
	setCfg("http://localhost", "test-key")
	resetFilesFlags()
	filesUploadFlags.course = "CS101"

	err := filesUploadCmd.RunE(filesUploadCmd, []string{"../../etc/passwd"})
	if err == nil || !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestFilesUpload_NotFound(t *testing.T) {
	setCfg("http://localhost", "test-key")
	resetFilesFlags()
	filesUploadFlags.course = "CS101"

	err := filesUploadCmd.RunE(filesUploadCmd, []string{"/nonexistent/file.txt"})
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

// ============================================================
// files download
// ============================================================

func TestFilesDownload_Success(t *testing.T) {
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/courses/CS101/files/items/i1/content": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Disposition", `attachment; filename="lecture.pdf"`)
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("pdf-content"))
		},
	})
	defer srv.Close()

	outPath := filepath.Join(t.TempDir(), "lecture.pdf")

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesDownloadFlags.course = "CS101"
	filesDownloadFlags.out = outPath

	var out bytes.Buffer
	filesDownloadCmd.SetOut(&out)
	if err := filesDownloadCmd.RunE(filesDownloadCmd, []string{"i1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if string(data) != "pdf-content" {
		t.Errorf("downloaded content = %q; want pdf-content", string(data))
	}
}

// ============================================================
// files rename
// ============================================================

func TestFilesRenameFile_Success(t *testing.T) {
	item := sampleFileItem("i1", "new-name.pdf")
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/courses/CS101/files/items/i1": func(w http.ResponseWriter, r *http.Request) {
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["displayName"] != "new-name.pdf" {
				http.Error(w, "wrong name", http.StatusBadRequest)
				return
			}
			item.DisplayName = req["displayName"]
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(item)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesRenameFlags.course = "CS101"
	filesRenameFlags.item = "i1"

	var out bytes.Buffer
	filesRenameCmd.SetOut(&out)
	if err := filesRenameCmd.RunE(filesRenameCmd, []string{"new-name.pdf"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "new-name.pdf") {
		t.Errorf("output = %q; want new-name.pdf", out.String())
	}
}

func TestFilesRenameFolder_Success(t *testing.T) {
	folder := sampleFolder("f1", "Updated Name")
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/courses/CS101/files/folders/f1": func(w http.ResponseWriter, r *http.Request) {
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			folder.Name = req["name"]
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(folder)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesRenameFlags.course = "CS101"
	filesRenameFlags.folder = "f1"

	var out bytes.Buffer
	filesRenameCmd.SetOut(&out)
	if err := filesRenameCmd.RunE(filesRenameCmd, []string{"Updated Name"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "Updated Name") {
		t.Errorf("output = %q; want Updated Name", out.String())
	}
}

func TestFilesRename_NoFlagError(t *testing.T) {
	setCfg("http://localhost", "test-key")
	resetFilesFlags()
	filesRenameFlags.course = "CS101"

	err := filesRenameCmd.RunE(filesRenameCmd, []string{"x"})
	if err == nil || !strings.Contains(err.Error(), "--item or --folder") {
		t.Errorf("expected flag error, got: %v", err)
	}
}

func TestFilesRename_BothFlagsError(t *testing.T) {
	setCfg("http://localhost", "test-key")
	resetFilesFlags()
	filesRenameFlags.course = "CS101"
	filesRenameFlags.item = "i1"
	filesRenameFlags.folder = "f1"

	err := filesRenameCmd.RunE(filesRenameCmd, []string{"x"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected exclusivity error, got: %v", err)
	}
}

// ============================================================
// files move
// ============================================================

func TestFilesMoveFile_ToFolder(t *testing.T) {
	folderID := "f2"
	item := sampleFileItem("i1", "notes.pdf")
	item.FolderID = &folderID
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/courses/CS101/files/items/i1": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(item)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesMoveFlags.course = "CS101"
	filesMoveFlags.item = "i1"
	filesMoveFlags.to = "f2"

	var out bytes.Buffer
	filesMoveCmd.SetOut(&out)
	if err := filesMoveCmd.RunE(filesMoveCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "f2") {
		t.Errorf("output = %q; want destination folder ID", out.String())
	}
}

func TestFilesMoveFolder_ToRoot(t *testing.T) {
	folder := sampleFolder("f1", "Lectures")
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"PATCH /api/v1/courses/CS101/files/folders/f1": func(w http.ResponseWriter, r *http.Request) {
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["parentId"] != "" {
				http.Error(w, "expected root", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(folder)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesMoveFlags.course = "CS101"
	filesMoveFlags.folder = "f1"
	// --to omitted => root

	var out bytes.Buffer
	filesMoveCmd.SetOut(&out)
	if err := filesMoveCmd.RunE(filesMoveCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "root") {
		t.Errorf("output = %q; want 'root'", out.String())
	}
}

// ============================================================
// files delete
// ============================================================

func TestFilesDeleteFile_Force(t *testing.T) {
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/courses/CS101/files/items/i1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesDeleteFlags.course = "CS101"
	filesDeleteFlags.item = "i1"
	filesDeleteFlags.force = true

	var out bytes.Buffer
	filesDeleteCmd.SetOut(&out)
	if err := filesDeleteCmd.RunE(filesDeleteCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "i1") {
		t.Errorf("output = %q; want item ID", out.String())
	}
}

func TestFilesDeleteFolder_Force(t *testing.T) {
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/courses/CS101/files/folders/f1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesDeleteFlags.course = "CS101"
	filesDeleteFlags.folder = "f1"
	filesDeleteFlags.force = true

	var out bytes.Buffer
	filesDeleteCmd.SetOut(&out)
	if err := filesDeleteCmd.RunE(filesDeleteCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "f1") {
		t.Errorf("output = %q; want folder ID", out.String())
	}
}

func TestFilesDelete_Aborted(t *testing.T) {
	setCfg("http://localhost", "test-key")
	resetFilesFlags()
	filesDeleteFlags.course = "CS101"
	filesDeleteFlags.item = "i1"
	filesDeleteFlags.force = false
	filesDeleteInput = strings.NewReader("n\n")

	var out bytes.Buffer
	filesDeleteCmd.SetOut(&out)
	if err := filesDeleteCmd.RunE(filesDeleteCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if !strings.Contains(out.String(), "Aborted") {
		t.Errorf("output = %q; want Aborted", out.String())
	}
}

func TestFilesDelete_NoFlagError(t *testing.T) {
	setCfg("http://localhost", "test-key")
	resetFilesFlags()
	filesDeleteFlags.course = "CS101"

	err := filesDeleteCmd.RunE(filesDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--item or --folder") {
		t.Errorf("expected flag error, got: %v", err)
	}
}

func TestFilesDelete_JSON(t *testing.T) {
	srv := newFilesServer(t, map[string]http.HandlerFunc{
		"DELETE /api/v1/courses/CS101/files/items/i1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFilesFlags()
	filesDeleteFlags.course = "CS101"
	filesDeleteFlags.item = "i1"
	filesDeleteFlags.force = true

	prev := globalFlags.jsonOut
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = prev }()

	var out bytes.Buffer
	filesDeleteCmd.SetOut(&out)
	if err := filesDeleteCmd.RunE(filesDeleteCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded["deleted"] != "i1" || decoded["type"] != "file" {
		t.Errorf("decoded = %+v", decoded)
	}
}

// strPtr is a helper for creating *string test fixtures.
func strPtr(s string) *string { return &s }
