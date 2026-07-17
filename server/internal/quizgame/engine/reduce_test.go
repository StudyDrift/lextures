package engine

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func baseState(phase Phase, idx, count int) State {
	return State{
		SessionID:     "s1",
		Status:        StatusLobby,
		Phase:         phase,
		Pacing:        PacingManual,
		QuestionIndex: idx,
		QuestionCount: count,
	}
}

func TestReduce_LobbyOpenStartsQ0(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	s := baseState(PhaseLobby, -1, 3)
	next, ev, err := Reduce(s, ActionOpen, now)
	if err != nil {
		t.Fatal(err)
	}
	if next.Phase != PhaseQuestionOpen || next.QuestionIndex != 0 {
		t.Fatalf("got phase=%s idx=%d", next.Phase, next.QuestionIndex)
	}
	if next.OpenedAt == nil || !next.OpenedAt.Equal(now) {
		t.Fatalf("opened_at=%v", next.OpenedAt)
	}
	if next.Status != StatusRunning {
		t.Fatalf("status=%s", next.Status)
	}
	if len(ev) != 1 || ev[0].Type != "question_open" {
		t.Fatalf("events=%v", ev)
	}
	withDL := ApplyDeadline(next, 20)
	if withDL.Deadline == nil || !withDL.Deadline.Equal(now.Add(20*time.Second)) {
		t.Fatalf("deadline=%v", withDL.Deadline)
	}
}

func TestReduce_IllegalOpenWhileOpen(t *testing.T) {
	s := baseState(PhaseQuestionOpen, 0, 2)
	_, _, err := Reduce(s, ActionOpen, time.Now().UTC())
	var illegal ErrIllegalTransition
	if !errors.As(err, &illegal) {
		t.Fatalf("want illegal, got %v", err)
	}
}

func TestReduce_LockRevealNextPodium(t *testing.T) {
	now := time.Now().UTC()
	s := baseState(PhaseQuestionOpen, 0, 1)
	opened := now.Add(-2 * time.Second)
	s.OpenedAt = &opened
	s.Status = StatusRunning

	locked, _, err := Reduce(s, ActionLock, now)
	if err != nil || locked.Phase != PhaseQuestionLocked {
		t.Fatalf("lock: %v phase=%s", err, locked.Phase)
	}
	revealed, _, err := Reduce(locked, ActionReveal, now)
	if err != nil || revealed.Phase != PhaseQuestionReveal {
		t.Fatalf("reveal: %v phase=%s", err, revealed.Phase)
	}
	lb, ev, err := Reduce(revealed, ActionNext, now)
	if err != nil || lb.Phase != PhaseLeaderboard {
		t.Fatalf("next→leaderboard: %v phase=%s ev=%v", err, lb.Phase, ev)
	}
	podium, ev, err := Reduce(lb, ActionNext, now)
	if err != nil || podium.Phase != PhasePodium {
		t.Fatalf("next→podium: %v phase=%s ev=%v", err, podium.Phase, ev)
	}
	ended, _, err := Reduce(podium, ActionEnd, now)
	if err != nil || ended.Phase != PhaseEnded || ended.Status != StatusEnded {
		t.Fatalf("end: %v phase=%s status=%s", err, ended.Phase, ended.Status)
	}
}

func TestReduce_NextOpensFollowingQuestion(t *testing.T) {
	now := time.Now().UTC()
	s := baseState(PhaseQuestionReveal, 0, 3)
	s.Status = StatusRunning
	lb, _, err := Reduce(s, ActionNext, now)
	if err != nil {
		t.Fatal(err)
	}
	if lb.Phase != PhaseLeaderboard {
		t.Fatalf("reveal next should enter leaderboard, got %s", lb.Phase)
	}
	next, _, err := Reduce(lb, ActionNext, now)
	if err != nil {
		t.Fatal(err)
	}
	if next.Phase != PhaseQuestionOpen || next.QuestionIndex != 1 {
		t.Fatalf("got phase=%s idx=%d", next.Phase, next.QuestionIndex)
	}
}

func TestResponseTiming_ServerClock(t *testing.T) {
	opened := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	deadline := opened.Add(20 * time.Second)
	recv := opened.Add(3200 * time.Millisecond)
	ms, late := ResponseTiming(opened, &deadline, recv)
	if late || ms < 3190 || ms > 3210 {
		t.Fatalf("ms=%d late=%v", ms, late)
	}
	lateRecv := opened.Add(25 * time.Second)
	_, late = ResponseTiming(opened, &deadline, lateRecv)
	if !late {
		t.Fatal("expected late")
	}
}

func TestReduce_HostDisconnectAndReconnect(t *testing.T) {
	now := time.Now().UTC()
	s := baseState(PhaseQuestionOpen, 1, 3)
	s.Status = StatusRunning
	paused, ev := ReduceHostDisconnect(s, now)
	if paused.Phase != PhaseWaitingForHost || paused.Status != StatusPaused || !paused.HostPaused {
		t.Fatalf("paused=%+v", paused)
	}
	if len(ev) != 1 || ev[0].Type != "host_disconnect" {
		t.Fatalf("ev=%v", ev)
	}
	disc := now
	resumed, _, ok := ReduceHostReconnect(paused, &disc, now.Add(30*time.Second), HostGraceDefault)
	if !ok || resumed.Phase != PhaseQuestionOpen || resumed.HostPaused {
		t.Fatalf("resume ok=%v state=%+v", ok, resumed)
	}
	_, _, ok = ReduceHostReconnect(paused, &disc, now.Add(2*time.Minute), HostGraceDefault)
	if ok {
		t.Fatal("expected grace expiry")
	}
}

func TestGradeAnswer_MC(t *testing.T) {
	q := SnapshotQuestion{
		QuestionType: "mc_single",
		Options: []Option{
			{ID: "a", Text: "A", IsCorrect: false},
			{ID: "b", Text: "B", IsCorrect: true},
		},
	}
	if !GradeAnswer(q, json.RawMessage(`{"optionId":"b"}`)) {
		t.Fatal("expected correct")
	}
	if GradeAnswer(q, json.RawMessage(`{"optionId":"a"}`)) {
		t.Fatal("expected incorrect")
	}
}

func TestStubPoints(t *testing.T) {
	if StubPoints("standard", true, "mc_single") != 1000 {
		t.Fatal("standard")
	}
	if StubPoints("double", true, "mc_single") != 2000 {
		t.Fatal("double")
	}
	if StubPoints("standard", false, "mc_single") != 0 {
		t.Fatal("incorrect")
	}
}

func TestGenerateJoinCode(t *testing.T) {
	c, err := GenerateJoinCode(6)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 6 {
		t.Fatalf("len=%d code=%q", len(c), c)
	}
	for _, ch := range c {
		if ch < '0' || ch > '9' {
			t.Fatalf("non-digit %q", c)
		}
	}
}

func TestToPublicQuestion_StripsCorrectness(t *testing.T) {
	q := SnapshotQuestion{
		Prompt:       "Q?",
		QuestionType: "mc_single",
		Options:      []Option{{ID: "a", Text: "A", IsCorrect: true}},
	}
	pub := ToPublicQuestion(q, 0)
	if pub.Options[0].IsCorrect {
		t.Fatal("correctness leaked")
	}
}
