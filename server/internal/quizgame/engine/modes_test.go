package engine

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeMode(t *testing.T) {
	if NormalizeMode("team") != ModeTeam {
		t.Fatal("team")
	}
	if NormalizeMode("bogus") != ModeLiveClassic {
		t.Fatal("default")
	}
}

func TestAggregateTeamScores_AverageUneven(t *testing.T) {
	members := []TeamMemberScore{
		{PlayerID: "p1", TeamID: "t1", TotalScore: 1000, ResponseMs: 100},
		{PlayerID: "p2", TeamID: "t1", TotalScore: 500, ResponseMs: 200},
		{PlayerID: "p3", TeamID: "t2", TotalScore: 900, ResponseMs: 50},
	}
	meta := map[string]struct{ Name, Color string }{
		"t1": {Name: "Alpha", Color: "#f00"},
		"t2": {Name: "Beta", Color: "#0f0"},
	}
	rows := AggregateTeamScores(members, meta, TeamAggregateAverage)
	if len(rows) != 2 {
		t.Fatalf("len=%d", len(rows))
	}
	// t1 avg = 750, t2 avg = 900 → t2 first
	if rows[0].TeamID != "t2" || rows[0].Score != 900 {
		t.Fatalf("expected t2@900 first, got %+v", rows[0])
	}
	if rows[1].TeamID != "t1" || rows[1].Score != 750 {
		t.Fatalf("expected t1@750 second, got %+v", rows[1])
	}
}

func TestAggregateTeamScores_Sum(t *testing.T) {
	members := []TeamMemberScore{
		{PlayerID: "p1", TeamID: "t1", TotalScore: 100},
		{PlayerID: "p2", TeamID: "t1", TotalScore: 100},
		{PlayerID: "p3", TeamID: "t2", TotalScore: 150},
	}
	meta := map[string]struct{ Name, Color string }{
		"t1": {Name: "A"}, "t2": {Name: "B"},
	}
	rows := AggregateTeamScores(members, meta, TeamAggregateSum)
	if rows[0].TeamID != "t1" || rows[0].Score != 200 {
		t.Fatalf("sum: %+v", rows[0])
	}
}

func TestApplyGradePolicy(t *testing.T) {
	scores := []int{100, 200, 150}
	if ApplyGradePolicy(scores, GradePolicyBest) != 200 {
		t.Fatal("best")
	}
	if ApplyGradePolicy(scores, GradePolicyLast) != 150 {
		t.Fatal("last")
	}
	avg := ApplyGradePolicy(scores, GradePolicyAverage)
	if avg < 149.9 || avg > 150.1 {
		t.Fatalf("avg=%v", avg)
	}
}

func TestCheckPlayWindow(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	tomorrow := now.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	if err := CheckPlayWindow(AssignmentWindow{OpensAt: &tomorrow}, now); err != ErrNotYetOpen {
		t.Fatalf("not yet open: %v", err)
	}
	if err := CheckPlayWindow(AssignmentWindow{OpensAt: &yesterday, ClosesAt: &yesterday}, now); err != ErrClosed {
		t.Fatalf("closed: %v", err)
	}
	if err := CheckPlayWindow(AssignmentWindow{OpensAt: &yesterday, DueAt: &yesterday, ClosesAt: &tomorrow}, now); err != nil {
		t.Fatalf("late but open: %v", err)
	}
	if !IsLate(AssignmentWindow{DueAt: &yesterday}, now) {
		t.Fatal("expected late")
	}
}

func TestEffectiveWindow_TimeMultiplier(t *testing.T) {
	opens := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	due := opens.Add(60 * time.Minute)
	base := AssignmentWindow{OpensAt: &opens, DueAt: &due}
	eff := EffectiveWindow(base, 1.5, opens)
	if eff.DueAt == nil {
		t.Fatal("nil due")
	}
	want := opens.Add(90 * time.Minute)
	if !eff.DueAt.Equal(want) {
		t.Fatalf("due=%v want=%v", eff.DueAt, want)
	}
}

func TestAggregatePacedProgress(t *testing.T) {
	// two players: one on Q1 (0-based), one finished
	buckets := AggregatePacedProgress(3, []int{1, 2}, []bool{false, true})
	if buckets[0].Reached != 2 {
		t.Fatalf("Q0 reached=%d", buckets[0].Reached)
	}
	if buckets[1].Reached != 2 {
		t.Fatalf("Q1 reached=%d", buckets[1].Reached)
	}
	if buckets[2].Finished != 1 {
		t.Fatalf("finished=%d", buckets[2].Finished)
	}
}

func TestMergeModeSettingsInto(t *testing.T) {
	raw, err := MergeModeSettingsInto(json.RawMessage(`{"foo":1}`), ModeTeam, &TeamConfig{TeamCount: 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["foo"].(float64) != 1 {
		t.Fatal("preserve base")
	}
	team, ok := m["team"].(map[string]any)
	if !ok || int(team["teamCount"].(float64)) != 3 {
		t.Fatalf("team=%v", m["team"])
	}
}

func TestAutoBalanceAssign(t *testing.T) {
	m := AutoBalanceAssign([]string{"a", "b", "c", "d"}, []string{"t1", "t2"})
	if m["a"] != "t1" || m["b"] != "t2" || m["c"] != "t1" {
		t.Fatalf("%v", m)
	}
}

func TestGuestsAllowed(t *testing.T) {
	if GuestsAllowed(ModeHomework) {
		t.Fatal("homework guests")
	}
	if !GuestsAllowed(ModeTeam) {
		t.Fatal("team guests")
	}
}
