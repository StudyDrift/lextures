package gradingagent

import (
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

func TestSubmissionAttemptableForAgent_includesTextEntryWhenEnabled(t *testing.T) {
	withFile := uuid.New()
	rows := []moduleassignmentsubmissions.SubmissionRow{
		{ID: uuid.New(), AttachmentFileID: &withFile},
		{ID: uuid.New(), BodyText: "Essay answer"},
		{ID: uuid.New()},
	}
	got := make([]moduleassignmentsubmissions.SubmissionRow, 0)
	for _, row := range rows {
		if SubmissionAttemptableForAgent(row, true) {
			got = append(got, row)
		}
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
}

func TestSubmissionAttemptableForAgent_textEntryDisabled(t *testing.T) {
	row := moduleassignmentsubmissions.SubmissionRow{ID: uuid.New(), BodyText: "typed only"}
	if SubmissionAttemptableForAgent(row, false) {
		t.Fatal("expected text-only row to be excluded when text entry disabled")
	}
}

func TestJoinSubmissions_preservesOrder(t *testing.T) {
	got := JoinSubmissions([]string{"first", "second"})
	if got != "first\n\nsecond" {
		t.Fatalf("got %q", got)
	}
}