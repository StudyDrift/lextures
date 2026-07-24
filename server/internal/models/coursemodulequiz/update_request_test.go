package coursemodulequiz

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUpdateModuleQuizRequestUnmarshalSparseAndNull(t *testing.T) {
	t.Parallel()

	var empty UpdateModuleQuizRequest
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatal(err)
	}
	if empty.HasUpdates() {
		t.Fatal("empty body should have no updates")
	}

	raw := `{
		"title":"Module 1 Checkpoint",
		"markdown":"## Intro",
		"dueAt":null,
		"timeLimitMinutes":null,
		"maxAttempts":3,
		"unlimitedAttempts":false,
		"gradeAttemptPolicy":"latest"
	}`
	var req UpdateModuleQuizRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if !req.HasUpdates() {
		t.Fatal("expected updates")
	}
	if req.Title == nil || *req.Title != "Module 1 Checkpoint" {
		t.Fatalf("title=%v", req.Title)
	}
	if req.Markdown == nil || *req.Markdown != "## Intro" {
		t.Fatalf("markdown=%v", req.Markdown)
	}
	if req.DueAt == nil || *req.DueAt != nil {
		t.Fatalf("dueAt should be present and clear, got %#v", req.DueAt)
	}
	if req.TimeLimitMinutes == nil || *req.TimeLimitMinutes != nil {
		t.Fatalf("timeLimitMinutes should be present and clear, got %#v", req.TimeLimitMinutes)
	}
	if req.MaxAttempts == nil || *req.MaxAttempts != 3 {
		t.Fatalf("maxAttempts=%v", req.MaxAttempts)
	}
	if req.Questions != nil {
		t.Fatal("questions should be omitted")
	}
	if err := req.ValidatePatch(); err != nil {
		t.Fatal(err)
	}

	setDue := `{"dueAt":"2026-07-24T12:00:00Z"}`
	var dueReq UpdateModuleQuizRequest
	if err := json.Unmarshal([]byte(setDue), &dueReq); err != nil {
		t.Fatal(err)
	}
	if dueReq.DueAt == nil || *dueReq.DueAt == nil {
		t.Fatal("expected dueAt value")
	}
	want := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	if !(*dueReq.DueAt).Equal(want) {
		t.Fatalf("dueAt=%v want=%v", **dueReq.DueAt, want)
	}
}

func TestUpdateModuleQuizRequestValidatePatch(t *testing.T) {
	t.Parallel()
	bad := "nope"
	req := UpdateModuleQuizRequest{GradeAttemptPolicy: &bad}
	if err := req.ValidatePatch(); err == nil {
		t.Fatal("expected invalid gradeAttemptPolicy")
	}
}
