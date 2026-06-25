package httpserver

import (
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

func TestGradableSubmissionsForAgent_keepsFilesAndTextEntry(t *testing.T) {
	withFile := uuid.New()
	rows := []moduleassignmentsubmissions.SubmissionRow{
		{ID: uuid.New(), AttachmentFileID: &withFile},
		{ID: uuid.New(), BodyText: "Online essay"},
		{ID: uuid.New()},
	}
	got := gradableSubmissionsForAgent(rows, true)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
}

func TestGradableSubmissionsForAgent_textEntryDisabled(t *testing.T) {
	rows := []moduleassignmentsubmissions.SubmissionRow{
		{ID: uuid.New(), BodyText: "typed only"},
	}
	if len(gradableSubmissionsForAgent(rows, false)) != 0 {
		t.Fatal("expected empty when text entry disabled")
	}
}

func TestGradableSubmissionsForAgent_EmptyInput(t *testing.T) {
	if gradableSubmissionsForAgent(nil, true) != nil {
		t.Fatal("expected nil for nil input")
	}
	if len(gradableSubmissionsForAgent([]moduleassignmentsubmissions.SubmissionRow{}, true)) != 0 {
		t.Fatal("expected empty slice")
	}
}