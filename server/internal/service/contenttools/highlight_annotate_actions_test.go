package contenttools

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/highlight_annotate"
)

func TestHandleHighlightAnnotateFilterNote_Empty(t *testing.T) {
	res, err := handleHighlightAnnotateFilterNote(ActionContext{
		ToolID: highlight_annotate.ID,
		Input:  json.RawMessage(`{"note":"  "}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["ok"] != true {
		t.Fatalf("result=%v", res.Result)
	}
}

func TestHandleHighlightAnnotateFilterNote_Block(t *testing.T) {
	res, err := handleHighlightAnnotateFilterNote(ActionContext{
		ToolID: highlight_annotate.ID,
		Input:  json.RawMessage(`{"note":"this contains fuck language"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "filtered" {
		t.Fatalf("expected filtered, got %v", res.Result)
	}
	if res.Result["preserveInput"] != true {
		t.Fatalf("preserveInput missing: %v", res.Result)
	}
}

func TestDeriveHighlightAnnotateStatus(t *testing.T) {
	cfg := json.RawMessage(`{"prompt":"p","tags":[{"id":"claim","label":"Claim","color":"#abc"}],"minAnnotations":2,"maxAnnotations":10}`)
	st := json.RawMessage(`{"v":1,"annotations":[{"id":"a1","tagId":"claim","quote":"q","anchor":{"prefix":"","suffix":"","approxOffset":0},"createdAt":"2026-01-01T00:00:00Z"}]}`)
	if got := DeriveHighlightAnnotateStatus(highlight_annotate.ID, cfg, st, StatusInProgress); got != StatusInProgress {
		t.Fatalf("partial=%s", got)
	}
	st2 := json.RawMessage(`{"v":1,"annotations":[{"id":"a1","tagId":"claim","quote":"q","anchor":{"prefix":"","suffix":"","approxOffset":0},"createdAt":"2026-01-01T00:00:00Z"},{"id":"a2","tagId":"claim","quote":"q2","anchor":{"prefix":"","suffix":"","approxOffset":1},"createdAt":"2026-01-01T00:00:01Z"}]}`)
	if got := DeriveHighlightAnnotateStatus(highlight_annotate.ID, cfg, st2, StatusInProgress); got != StatusCompleted {
		t.Fatalf("complete=%s", got)
	}
	if got := DeriveHighlightAnnotateStatus("other", cfg, st2, StatusInProgress); got != "" {
		t.Fatalf("other tool=%s", got)
	}
}
