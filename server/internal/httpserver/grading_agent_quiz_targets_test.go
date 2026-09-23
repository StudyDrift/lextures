package httpserver

import (
	"testing"

	"github.com/google/uuid"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

func quizAttempt(student uuid.UUID, id uuid.UUID, n int32, manual bool) quizattempts.AttemptListRow {
	return quizattempts.AttemptListRow{
		ID:                 id,
		StudentUserID:      student,
		StudentDisplayName: "Student",
		AttemptNumber:      n,
		NeedsManualGrading: manual,
	}
}

func TestLatestQuizAttemptsKeepsHighestNumber(t *testing.T) {
	student := uuid.New()
	first := uuid.New()
	second := uuid.New()
	rows := []quizattempts.AttemptListRow{
		quizAttempt(student, first, 1, true),
		quizAttempt(student, second, 2, false),
	}
	got := latestQuizAttempts(rows)
	if len(got) != 1 || got[0].ID != second {
		t.Fatalf("latest = %+v, want attempt 2", got)
	}
}

func TestLatestQuizAttemptsTieKeepsLaterRow(t *testing.T) {
	student := uuid.New()
	first := uuid.New()
	second := uuid.New()
	rows := []quizattempts.AttemptListRow{
		quizAttempt(student, first, 2, true),
		quizAttempt(student, second, 2, false),
	}
	got := latestQuizAttempts(rows)
	if len(got) != 1 || got[0].ID != second {
		t.Fatalf("tie latest = %+v, want later row", got)
	}
}

func TestSelectQuizAttemptTargetsUngraded(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	rows := []quizattempts.AttemptListRow{
		quizAttempt(a, uuid.New(), 1, true),
		quizAttempt(b, uuid.New(), 1, false),
	}
	got, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeUngraded, "", false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].StudentUserID != a {
		t.Fatalf("ungraded = %+v", got)
	}
}

func TestSelectQuizAttemptTargetsAllRequiresOverwrite(t *testing.T) {
	rows := []quizattempts.AttemptListRow{quizAttempt(uuid.New(), uuid.New(), 1, true)}
	if _, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeAll, "", false, nil, nil); err == nil {
		t.Fatal("expected overwrite error")
	}
	got, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeAll, "", true, nil, nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("all overwrite: %v %+v", err, got)
	}
}

func TestSelectQuizAttemptTargetsCurrentAndStudentFilter(t *testing.T) {
	student := uuid.New()
	other := uuid.New()
	attemptID := uuid.New()
	rows := []quizattempts.AttemptListRow{
		quizAttempt(student, attemptID, 1, true),
		quizAttempt(other, uuid.New(), 1, true),
	}
	allow := map[uuid.UUID]struct{}{student: {}}
	got, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeCurrent, attemptID.String(), false, allow, nil)
	if err != nil || len(got) != 1 || got[0].ID != attemptID {
		t.Fatalf("current: %v %+v", err, got)
	}
	if _, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeCurrent, "", false, nil, nil); err == nil {
		t.Fatal("expected missing submission id error")
	}
	hidden := uuid.New()
	if _, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeCurrent, hidden.String(), false, allow, nil); err == nil {
		t.Fatal("expected missing attempt error")
	}
}

func TestSelectQuizAttemptTargetsExplicitAttemptIDs(t *testing.T) {
	student := uuid.New()
	older := uuid.New()
	newer := uuid.New()
	rows := []quizattempts.AttemptListRow{
		quizAttempt(student, newer, 2, false),
		quizAttempt(student, older, 1, true),
	}
	only := map[uuid.UUID]struct{}{older: {}}
	got, err := selectQuizAttemptTargets(rows, gradingagentrepo.RunScopeUngraded, "", false, nil, only)
	if err != nil || len(got) != 1 || got[0].ID != older {
		t.Fatalf("explicit id = %v %+v", err, got)
	}
}

func TestLabelsForQuizAttempts(t *testing.T) {
	id := uuid.New()
	missing := uuid.New()
	rows := []quizattempts.AttemptListRow{{
		ID:                 id,
		StudentUserID:      uuid.New(),
		StudentDisplayName: "Ada Lovelace",
	}}
	labels := labelsForQuizAttempts(rows, []uuid.UUID{id, missing})
	if labels[id] != "Ada Lovelace" {
		t.Fatalf("label = %q", labels[id])
	}
	if labels[missing] != missing.String()[:8] {
		t.Fatalf("missing label = %q", labels[missing])
	}
}
