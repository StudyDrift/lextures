package httpserver

import (
	"strings"
	"testing"
)

func TestCanvasAssignmentSubmissionImportable(t *testing.T) {
	if !canvasAssignmentSubmissionImportable(map[string]any{"workflow_state": "submitted"}) {
		t.Fatal("submitted workflow should import")
	}
	if !canvasAssignmentSubmissionImportable(map[string]any{"body": "<p>Hello</p>"}) {
		t.Fatal("text body should import")
	}
	if canvasAssignmentSubmissionImportable(map[string]any{"workflow_state": "unsubmitted"}) {
		t.Fatal("unsubmitted without content should not import")
	}
}

func TestCanvasSubmissionTextForImport(t *testing.T) {
	text, ok := canvasSubmissionTextForImport(map[string]any{
		"body": "<p>Answer</p>",
		"url":  "https://example.com/doc",
	})
	if !ok {
		t.Fatal("expected text")
	}
	if !strings.Contains(text, "Answer") || !strings.Contains(text, "https://example.com/doc") {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestCanvasEffectiveSubmissionPayloadUsesHistory(t *testing.T) {
	sub := map[string]any{
		"workflow_state": "submitted",
		"submission_history": []any{
			map[string]any{"body": "<p>Latest answer</p>"},
		},
	}
	text, ok := canvasSubmissionTextForImport(sub)
	if !ok || !strings.Contains(text, "Latest answer") {
		t.Fatalf("expected history body, got ok=%v text=%q", ok, text)
	}
}

func TestCanvasSubmissionAttachments(t *testing.T) {
	sub := map[string]any{
		"attachments": []any{
			map[string]any{"id": float64(1), "filename": "a.pdf"},
			map[string]any{"id": float64(2), "filename": "b.pdf"},
		},
	}
	atts := canvasSubmissionAttachments(sub)
	if len(atts) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(atts))
	}
	att := canvasFirstSubmissionAttachment(sub)
	if att == nil || int64At(att, "id") != 1 {
		t.Fatalf("expected first attachment, got %#v", att)
	}
}

func TestCanvasAttachmentByteSize(t *testing.T) {
	att := map[string]any{"size": float64(32 << 20)}
	if got := canvasAttachmentByteSize(att); got != 32<<20 {
		t.Fatalf("expected 32 MiB, got %d", got)
	}
	if canvasAttachmentByteSize(nil) != 0 {
		t.Fatal("nil attachment should report size 0")
	}
}

func TestCanvasMaxImportedSubmissionFileBytesAllowsLargeVideo(t *testing.T) {
	// Regression: Canvas video submissions (e.g. 32 MB) must not hit the old 20 MB cap.
	if canvasMaxImportedSubmissionFileBytes < 32<<20 {
		t.Fatalf("import cap %d is below 32 MiB", canvasMaxImportedSubmissionFileBytes)
	}
}
