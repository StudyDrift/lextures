package gradingagent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func TestLoadSubmissionTextForSubmission_RequiresAttachment(t *testing.T) {
	svc := &Service{}
	_, err := svc.LoadSubmissionTextForSubmission(context.Background(), "C-TEST", &moduleassignmentsubmissions.SubmissionRow{})
	if err == nil {
		t.Fatal("expected error without attachment")
	}
}

func TestReadSubmissionBlob_StorageDriverKey(t *testing.T) {
	dir := t.TempDir()
	storage := &filestorage.LocalDriver{Root: dir}
	key := "submissions/C-TEST01/abc.txt"
	content := []byte("hello submission")
	ctx := context.Background()
	if err := storage.PutObject(ctx, key, bytes.NewReader(content), int64(len(content)), "text/plain"); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	svc := &Service{Storage: storage}
	row := &coursefiles.Row{StorageKey: key}
	got, err := svc.readSubmissionBlob(ctx, "C-TEST01", row)
	if err != nil {
		t.Fatalf("readSubmissionBlob: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content = %q want %q", got, content)
	}
}

func TestReadSubmissionBlob_BlobDiskPathFallback(t *testing.T) {
	root := t.TempDir()
	courseCode := "C-TEST01"
	key := "submissions/C-TEST01/legacy.txt"
	content := []byte("legacy disk")
	p := coursefiles.BlobDiskPath(root, courseCode, key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &Service{FilesRoot: root}
	got, err := svc.readSubmissionBlob(context.Background(), courseCode, &coursefiles.Row{StorageKey: key})
	if err != nil {
		t.Fatalf("readSubmissionBlob: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content = %q want %q", got, content)
	}
}

func TestScore_OpenRouterIntegration(t *testing.T) {
	modelJSON := `{"total":8,"comment":"Solid work.","confidence":0.75,"rubric":{}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		quoted, _ := json.Marshal(modelJSON)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + string(quoted) + `}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"cost":0.01}}`))
	}))
	defer srv.Close()

	svc := &Service{Client: openrouter.NewClientWithBaseURL("test-key", srv.URL+"/v1")}
	result, err := svc.Score(context.Background(), ScoreRequest{
		InstructorPrompt: "Grade fairly.",
		SubmissionText:   "My essay argues for renewable energy.",
		ModelID:          "test/model",
		MaxPoints:        10,
	})
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if result.Output.TotalPoints != 8 {
		t.Fatalf("total=%v want 8", result.Output.TotalPoints)
	}
	if result.Output.Comment != "Solid work." {
		t.Fatalf("comment=%q", result.Output.Comment)
	}
}