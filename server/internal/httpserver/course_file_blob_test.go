package httpserver

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

func TestReadCourseFileRowBytes_StorageDriverKey(t *testing.T) {
	dir := t.TempDir()
	storage := &filestorage.LocalDriver{Root: dir}
	key := "submissions/import/C-TEST01/abc.txt"
	content := []byte("hello submission")
	ctx := context.Background()
	if err := storage.PutObject(ctx, key, bytes.NewReader(content), int64(len(content)), "text/plain"); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	d := Deps{Storage: storage}
	row := &coursefiles.Row{StorageKey: key, MimeType: "text/plain"}
	got, err := d.readCourseFileRowBytes(ctx, "C-TEST01", row)
	if err != nil {
		t.Fatalf("readCourseFileRowBytes: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content = %q want %q", got, content)
	}
}

func TestReadCourseFileRowBytes_BlobDiskPathFallback(t *testing.T) {
	root := t.TempDir()
	courseCode := "C-TEST01"
	key := "submissions/import/C-TEST01/legacy.txt"
	content := []byte("legacy disk")
	p := coursefiles.BlobDiskPath(root, courseCode, key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}

	d := Deps{Storage: nil, Config: config.Config{CourseFilesRoot: root}}
	row := &coursefiles.Row{StorageKey: key}
	got, err := d.readCourseFileRowBytes(context.Background(), courseCode, row)
	if err != nil {
		t.Fatalf("readCourseFileRowBytes: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content = %q want %q", got, content)
	}
}
