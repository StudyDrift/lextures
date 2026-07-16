package gradingagent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// stubCompleter is a minimal aiprovider.ScopedCompleter test double.
type stubCompleter struct {
	completeFunc func(ctx context.Context, modelOverride string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error)
	calls        int
}

func (s *stubCompleter) Complete(ctx context.Context, modelOverride string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	s.calls++
	if s.completeFunc != nil {
		return s.completeFunc(ctx, modelOverride, messages, opts...)
	}
	return aiprovider.ChatResult{}, aiprovider.CallMeta{}, nil
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

func TestScore_UsesScopedCompleter(t *testing.T) {
	modelJSON := `{"total":8,"comment":"Solid work.","confidence":0.75,"rubric":{}}`
	stub := &stubCompleter{
		completeFunc: func(ctx context.Context, modelOverride string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
			if modelOverride != "test/model" {
				t.Fatalf("modelOverride=%q", modelOverride)
			}
			return aiprovider.ChatResult{
				Text: modelJSON,
				Usage: aiprovider.UsageInfo{
					PromptTokens:     10,
					CompletionTokens: 5,
					CostUSD:          0.01,
				},
			}, aiprovider.CallMeta{Provider: aiprovider.ProviderAnthropic, ModelID: "test/model"}, nil
		},
	}

	svc := &Service{AI: stub}
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
	if result.CallMeta.Provider != aiprovider.ProviderAnthropic {
		t.Fatalf("callMeta.Provider=%q", result.CallMeta.Provider)
	}
	if stub.calls != 1 {
		t.Fatalf("calls=%d want 1", stub.calls)
	}
}

func TestScore_RetriesWithoutJSONModeOn400(t *testing.T) {
	modelJSON := `{"total":5,"comment":"OK.","confidence":0.5,"rubric":{}}`
	attempts := 0
	stub := &stubCompleter{
		completeFunc: func(ctx context.Context, modelOverride string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
			attempts++
			if len(opts) > 0 && opts[0].JSONMode {
				return aiprovider.ChatResult{}, aiprovider.CallMeta{}, fmt.Errorf("aiprovider: status 400: response_format not supported")
			}
			return aiprovider.ChatResult{Text: modelJSON}, aiprovider.CallMeta{}, nil
		},
	}

	svc := &Service{AI: stub}
	result, err := svc.Score(context.Background(), ScoreRequest{
		InstructorPrompt: "Grade fairly.",
		SubmissionText:   "Essay text.",
		ModelID:          "test/model",
		MaxPoints:        10,
	})
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts=%d want 2", attempts)
	}
	if result.Output.TotalPoints != 5 {
		t.Fatalf("total=%v want 5", result.Output.TotalPoints)
	}
}

func TestScore_NoAIConfigured(t *testing.T) {
	svc := &Service{}
	_, err := svc.Score(context.Background(), ScoreRequest{
		InstructorPrompt: "Grade fairly.",
		SubmissionText:   "Essay text.",
		ModelID:          "test/model",
	})
	if err == nil {
		t.Fatal("expected error when AI is not configured")
	}
}
