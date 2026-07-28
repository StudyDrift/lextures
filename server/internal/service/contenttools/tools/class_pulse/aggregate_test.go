package class_pulse

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestBuildRoundAggregate_RoleExclusionAndSuppression(t *testing.T) {
	sec := uuid.New()
	other := uuid.New()
	cfg := DefaultConfig()
	cfg.MinRespondents = 5
	cfg.ShowPercentages = true

	rows := []VoteRow{
		{Role: "student", SectionID: &sec, State: State{Votes: []Vote{{Round: 1, OptionID: "a"}}}},
		{Role: "student", SectionID: &sec, State: State{Votes: []Vote{{Round: 1, OptionID: "a"}}}},
		{Role: "student", SectionID: &sec, State: State{Votes: []Vote{{Round: 1, OptionID: "b"}}}},
		{Role: "instructor", SectionID: &sec, State: State{Votes: []Vote{{Round: 1, OptionID: "a"}}}},
		{Role: "student", SectionID: &other, State: State{Votes: []Vote{{Round: 1, OptionID: "b"}}}},
	}

	got := BuildRoundAggregate(rows, 1, cfg, &sec)
	if !got.Suppressed || got.Learners != 3 {
		t.Fatalf("want suppressed with 3 learners, got %#v", got)
	}

	rows = append(rows,
		VoteRow{Role: "student", SectionID: &sec, State: State{Votes: []Vote{{Round: 1, OptionID: "b"}}}},
		VoteRow{Role: "student", SectionID: &sec, State: State{Votes: []Vote{{Round: 1, OptionID: "a"}}}},
	)
	got = BuildRoundAggregate(rows, 1, cfg, &sec)
	if got.Suppressed || got.Learners != 5 {
		t.Fatalf("want 5 learners unsuppressed, got %#v", got)
	}
	var aCount, bCount int
	for _, o := range got.Options {
		if o.OptionID == "a" {
			aCount = o.Count
		}
		if o.OptionID == "b" {
			bCount = o.Count
		}
	}
	if aCount != 3 || bCount != 2 {
		t.Fatalf("counts a=%d b=%d options=%#v", aCount, bCount, got.Options)
	}
}

func TestBuildShift(t *testing.T) {
	rows := []VoteRow{
		{Role: "student", State: State{Votes: []Vote{
			{Round: 1, OptionID: "a"}, {Round: 2, OptionID: "b"},
		}}},
		{Role: "student", State: State{Votes: []Vote{
			{Round: 1, OptionID: "a"}, {Round: 2, OptionID: "a"},
		}}},
		{Role: "ta", State: State{Votes: []Vote{
			{Round: 1, OptionID: "a"}, {Round: 2, OptionID: "b"},
		}}},
	}
	shift := BuildShift(rows, nil)
	if len(shift) != 2 {
		t.Fatalf("shift: %#v", shift)
	}
}

func TestShouldRevealCorrect(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CorrectOptionID = "a"
	cfg.RevealCorrect = RevealAfterRevote
	cfg.AllowSecondVote = true
	st := State{Votes: []Vote{{Round: 1, OptionID: "b"}}}
	if ShouldRevealCorrect(cfg, st) {
		t.Fatal("should not reveal after first vote when after_revote")
	}
	st.Votes = append(st.Votes, Vote{Round: 2, OptionID: "a"})
	if !ShouldRevealCorrect(cfg, st) {
		t.Fatal("should reveal after revote")
	}
}

func TestGuardStatePut(t *testing.T) {
	cur, _ := json.Marshal(State{Votes: []Vote{{Round: 1, OptionID: "a", At: "t"}}})
	blocked, _ := GuardStatePut(cur, []byte(`{"draft":{"optionId":"b"}}`))
	if blocked {
		t.Fatal("draft-only PUT should be allowed")
	}
	next, _ := json.Marshal(State{Votes: []Vote{{Round: 1, OptionID: "b", At: "t"}}})
	blocked, msg := GuardStatePut(cur, next)
	if !blocked || msg == "" {
		t.Fatal("vote mutation must be blocked")
	}
}
