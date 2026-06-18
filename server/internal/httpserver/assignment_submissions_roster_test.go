package httpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

func TestBuildAssignmentRosterEntries_includesEnrolledWithoutSubmission(t *testing.T) {
	alice := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	bob := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	carol := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	students := []struct {
		UserID      uuid.UUID
		DisplayName string
	}{
		{UserID: alice, DisplayName: "Alice"},
		{UserID: bob, DisplayName: "Bob"},
		{UserID: carol, DisplayName: "Carol"},
	}
	submissions := []moduleassignmentsubmissions.SubmissionRow{
		{ID: uuid.New(), SubmittedBy: bob},
	}
	entries := buildAssignmentRosterEntries(students, submissions)
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	if entries[0].UserID != alice || entries[0].Submission != nil {
		t.Fatalf("Alice entry = %+v, want roster row without submission", entries[0])
	}
	if entries[1].UserID != bob || entries[1].Submission == nil {
		t.Fatalf("Bob entry = %+v, want submission attached", entries[1])
	}
	if entries[2].UserID != carol || entries[2].Submission != nil {
		t.Fatalf("Carol entry = %+v, want roster row without submission", entries[2])
	}
}

func TestSubmissionMatchesGradedFilter(t *testing.T) {
	if !submissionMatchesGradedFilter(true, moduleassignmentsubmissions.GradedFilterGraded) {
		t.Fatal("graded student should match graded filter")
	}
	if submissionMatchesGradedFilter(false, moduleassignmentsubmissions.GradedFilterGraded) {
		t.Fatal("ungraded student should not match graded filter")
	}
	if !submissionMatchesGradedFilter(false, moduleassignmentsubmissions.GradedFilterUngraded) {
		t.Fatal("ungraded student should match ungraded filter")
	}
	if !submissionMatchesGradedFilter(true, moduleassignmentsubmissions.GradedFilterAll) {
		t.Fatal("all filter should include graded students")
	}
}