package quizgame

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/quizgame/engine"
)

func TestComputeReportAggregates_AvgMedianCorrectPct(t *testing.T) {
	sess := &Session{
		ID: "11111111-1111-1111-1111-111111111111",
		KitSnapshot: engine.KitSnapshot{
			Questions: []engine.SnapshotQuestion{
				{Prompt: "Q1", TimeLimitSeconds: 20},
				{Prompt: "Q2", TimeLimitSeconds: 20},
			},
		},
	}
	players := []Player{
		{ID: "p1", TotalScore: 100},
		{ID: "p2", TotalScore: 200},
		{ID: "p3", TotalScore: 300},
	}
	responses := []Response{
		{PlayerID: "p1", QuestionIndex: 0, IsCorrect: true, ResponseMs: 1000, Answer: json.RawMessage(`{"selectedOptionIds":["b"]}`)},
		{PlayerID: "p2", QuestionIndex: 0, IsCorrect: false, ResponseMs: 2000, Answer: json.RawMessage(`{"selectedOptionIds":["a"]}`)},
		{PlayerID: "p1", QuestionIndex: 1, IsCorrect: false, ResponseMs: 3000, Answer: json.RawMessage(`{"selectedOptionIds":["a"]}`)},
		{PlayerID: "p2", QuestionIndex: 1, IsCorrect: false, ResponseMs: 4000, Answer: json.RawMessage(`{"selectedOptionIds":["a"]}`)},
		{PlayerID: "p3", QuestionIndex: 1, IsCorrect: true, ResponseMs: 1500, Answer: json.RawMessage(`{"selectedOptionIds":["b"]}`)},
	}
	rep := ComputeReportAggregates(sess, players, responses)
	if rep.PlayerCount != 3 || rep.AnsweredCount != 3 {
		t.Fatalf("players=%d answered=%d", rep.PlayerCount, rep.AnsweredCount)
	}
	if rep.ScoreAvg == nil || *rep.ScoreAvg != 200 {
		t.Fatalf("avg=%v want 200", rep.ScoreAvg)
	}
	if rep.ScoreMedian == nil || *rep.ScoreMedian != 200 {
		t.Fatalf("median=%v want 200", rep.ScoreMedian)
	}
	if rep.ScoreMax == nil || *rep.ScoreMax != 300 {
		t.Fatalf("max=%v want 300", rep.ScoreMax)
	}
	if len(rep.PerQuestion) != 2 {
		t.Fatalf("perQ len=%d", len(rep.PerQuestion))
	}
	if rep.PerQuestion[0].CorrectPct != 50 {
		t.Fatalf("q0 correctPct=%v want 50", rep.PerQuestion[0].CorrectPct)
	}
	if rep.PerQuestion[0].AvgMs != 1500 {
		t.Fatalf("q0 avgMs=%v want 1500", rep.PerQuestion[0].AvgMs)
	}
	// Q1 harder (33.33% vs 50%) → hardestRank 1
	if rep.PerQuestion[1].HardestRank != 1 {
		t.Fatalf("q1 hardestRank=%d want 1", rep.PerQuestion[1].HardestRank)
	}
	if rep.PerQuestion[0].HardestRank != 2 {
		t.Fatalf("q0 hardestRank=%d want 2", rep.PerQuestion[0].HardestRank)
	}
}

func TestReportsMatch_Deterministic(t *testing.T) {
	sess := &Session{
		ID: "11111111-1111-1111-1111-111111111111",
		KitSnapshot: engine.KitSnapshot{
			Questions: []engine.SnapshotQuestion{{Prompt: "Q1"}},
		},
	}
	players := []Player{{ID: "p1", TotalScore: 50}, {ID: "p2", TotalScore: 150}}
	responses := []Response{
		{PlayerID: "p1", QuestionIndex: 0, IsCorrect: true, ResponseMs: 500, Answer: json.RawMessage(`{"value":1}`)},
		{PlayerID: "p2", QuestionIndex: 0, IsCorrect: false, ResponseMs: 800, Answer: json.RawMessage(`{"value":0}`)},
	}
	a := ComputeReportAggregates(sess, players, responses)
	b := ComputeReportAggregates(sess, players, responses)
	if !ReportsMatch(&a, &b) {
		t.Fatal("expected recomputation to match")
	}
}

func TestMapPlayerGrade_Mappings(t *testing.T) {
	// participation: answered 2/4 = 50% with threshold 50 → full points
	got := MapPlayerGrade(MappingParticipation, 10, 50, 0, 0, 2, 4, 0)
	if got != 10 {
		t.Fatalf("participation=%v want 10", got)
	}
	got = MapPlayerGrade(MappingParticipation, 10, 75, 0, 0, 2, 4, 0)
	if got != 0 {
		t.Fatalf("participation below threshold=%v want 0", got)
	}
	// percent correct: 3/4 → 7.5 of 10
	got = MapPlayerGrade(MappingPercentCorrect, 10, 50, 0, 0, 4, 4, 3)
	if got != 7.5 {
		t.Fatalf("percent=%v want 7.5", got)
	}
	// raw points scaled to max
	got = MapPlayerGrade(MappingRawPoints, 100, 50, 500, 1000, 0, 0, 0)
	if got != 50 {
		t.Fatalf("raw=%v want 50", got)
	}
}

func TestNormalizeMapping(t *testing.T) {
	m, err := NormalizeMapping("")
	if err != nil || m != MappingParticipation {
		t.Fatalf("default=%v err=%v", m, err)
	}
	if _, err := NormalizeMapping("nope"); err == nil {
		t.Fatal("expected invalid mapping error")
	}
}
