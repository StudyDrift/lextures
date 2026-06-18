package httpserver

import (
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

func TestGradableSubmissionsForAgent_KeepsOnlyRowsWithFiles(t *testing.T) {
	withFile := uuid.New()
	rows := []moduleassignmentsubmissions.SubmissionRow{
		{ID: uuid.New(), AttachmentFileID: &withFile},
		{ID: uuid.New(), AttachmentFileID: nil},
		{ID: uuid.New(), AttachmentFileID: func() *uuid.UUID { id := uuid.New(); return &id }()},
	}
	got := gradableSubmissionsForAgent(rows)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].AttachmentFileID == nil || got[1].AttachmentFileID == nil {
		t.Fatal("expected file attachments on all kept rows")
	}
}

func TestGradableSubmissionsForAgent_EmptyInput(t *testing.T) {
	if gradableSubmissionsForAgent(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
	if len(gradableSubmissionsForAgent([]moduleassignmentsubmissions.SubmissionRow{})) != 0 {
		t.Fatal("expected empty slice")
	}
}