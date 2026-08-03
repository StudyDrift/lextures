package courseexportimport

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
)

func TestDecodeFileContent_RoundTrip(t *testing.T) {
	raw := []byte("hello course file")
	b64 := base64.StdEncoding.EncodeToString(raw)
	got, err := decodeFileContent("test", b64, int64(len(raw)))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(got) != string(raw) {
		t.Fatalf("got %q want %q", got, raw)
	}
}

func TestDecodeFileContent_Empty(t *testing.T) {
	got, err := decodeFileContent("test", "", 0)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != nil {
		t.Fatalf("want nil for empty content, got %v", got)
	}
}

func TestDecodeFileContent_InvalidBase64(t *testing.T) {
	_, err := decodeFileContent("test", "!!!not-base64!!!", 0)
	if !IsInvalidInput(err) {
		t.Fatalf("want InvalidInput, got %v", err)
	}
}

func TestRewriteCourseFileURLsInBundle(t *testing.T) {
	pageID := uuid.New()
	hero := "/api/v1/courses/C-SRC/course-files/abc/content"
	ex := &Bundle{
		CourseCode: "C-SRC",
		Course: courseexport.CourseExportSnapshot{
			HeroImageURL: &hero,
		},
		Syllabus: nil,
		ContentPages: map[uuid.UUID]courseexport.ExportedContentPageBody{
			pageID: {Markdown: "![x](/api/v1/courses/C-SRC/files/items/xyz/content)"},
		},
		Assignments: map[uuid.UUID]courseexport.ExportedAssignmentBody{},
		Quizzes:     map[uuid.UUID]courseexport.ExportedQuizBody{},
	}
	rewriteCourseFileURLsInBundle(ex, "C-SRC", "C-DST")
	if ex.Course.HeroImageURL == nil || *ex.Course.HeroImageURL != "/api/v1/courses/C-DST/course-files/abc/content" {
		t.Fatalf("hero = %v", ex.Course.HeroImageURL)
	}
	if got := ex.ContentPages[pageID].Markdown; got != "![x](/api/v1/courses/C-DST/files/items/xyz/content)" {
		t.Fatalf("markdown = %q", got)
	}
}

func TestRewriteCourseFileURLsInBundle_SameCodeNoop(t *testing.T) {
	hero := "/api/v1/courses/C-A/course-files/abc/content"
	ex := &Bundle{
		Course: courseexport.CourseExportSnapshot{HeroImageURL: &hero},
	}
	rewriteCourseFileURLsInBundle(ex, "C-A", "C-A")
	if *ex.Course.HeroImageURL != hero {
		t.Fatalf("unexpected rewrite: %s", *ex.Course.HeroImageURL)
	}
}

func TestBlobOptions_WriteReadCourseFile(t *testing.T) {
	root := t.TempDir()
	opts := BlobOptions{FilesRoot: root}
	courseCode := "C-TEST01"
	key := "files/" + courseCode + "/" + uuid.New().String() + ".txt"
	data := []byte("embedded image bytes")
	ctx := context.Background()
	if err := opts.writeCourseFileBlob(ctx, courseCode, key, "text/plain", data); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Local layout uses BlobDiskPath (basename only).
	wantPath := coursefiles.BlobDiskPath(root, courseCode, key)
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected blob at %s: %v", wantPath, err)
	}
	got, err := opts.readCourseFileBlob(ctx, courseCode, key)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("got %q want %q", got, data)
	}
}

func TestBlobOptions_WriteReadFileItem(t *testing.T) {
	root := t.TempDir()
	opts := BlobOptions{FilesRoot: root}
	courseCode := "C-TEST01"
	key := "managed-files/" + courseCode + "/" + uuid.New().String() + ".pdf"
	data := []byte("%PDF-1.4 test")
	ctx := context.Background()
	if err := opts.writeFileItemBlob(ctx, courseCode, key, "application/pdf", data); err != nil {
		t.Fatalf("write: %v", err)
	}
	wantPath := filepath.Join(root, courseCode, filepath.FromSlash(key))
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected blob at %s: %v", wantPath, err)
	}
	got, err := opts.readFileItemBlob(ctx, courseCode, key)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("got %q want %q", got, data)
	}
}

func TestValidateCourseFilesExport_OK(t *testing.T) {
	fileID := uuid.New()
	folderID := uuid.New()
	itemID := uuid.New()
	ex := &Bundle{
		FormatVersion: 1,
		CourseCode:    "C-SRC",
		CourseFiles: []courseexport.ExportedCourseFile{
			{
				ID:               fileID,
				OriginalFilename: "pic.png",
				MimeType:         "image/png",
				ByteSize:         1,
				ContentBase64:    base64.StdEncoding.EncodeToString([]byte("x")),
			},
		},
		FileFolders: []courseexport.ExportedFileFolder{
			{ID: folderID, Name: "Handouts"},
		},
		FileItems: []courseexport.ExportedFileItem{
			{
				ID:               itemID,
				FolderID:         &folderID,
				OriginalFilename: "notes.pdf",
				DisplayName:      "Notes",
				MimeType:         "application/pdf",
				ByteSize:         1,
				ContentBase64:    base64.StdEncoding.EncodeToString([]byte("y")),
			},
		},
	}
	if err := validateCourseFilesExport(ex); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateCourseFilesExport_DuplicateID(t *testing.T) {
	id := uuid.New()
	ex := &Bundle{
		CourseFiles: []courseexport.ExportedCourseFile{
			{ID: id, OriginalFilename: "a.png"},
			{ID: id, OriginalFilename: "b.png"},
		},
	}
	err := validateCourseFilesExport(ex)
	if !IsInvalidInput(err) {
		t.Fatalf("want InvalidInput, got %v", err)
	}
}
