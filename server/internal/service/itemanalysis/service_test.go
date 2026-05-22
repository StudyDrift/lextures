package itemanalysis

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
)

var (
	quizID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	nowTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

func aid(i int) uuid.UUID {
	s := uuid.UUID{}
	s[15] = byte(i)
	return s
}

func boolPtr(b bool) *bool      { return &b }
func intPtr(i int) *int         { return &i }
func f64Ptr(f float64) *float64 { return &f }

// makeRow builds an AttemptResponseRow. attemptScorePct is the attempt-level score_percent (0-100).
func makeRow(attemptIdx, qi int, qtype string, correct bool, choice *int, ptsAwarded, maxPts, attemptScorePct float64) repoitemanalysis.AttemptResponseRow {
	return repoitemanalysis.AttemptResponseRow{
		AttemptID:     aid(attemptIdx),
		ScorePercent:  &attemptScorePct,
		QuestionIndex: qi,
		QuestionType:  qtype,
		IsCorrect:     boolPtr(correct),
		ChoiceIndex:   choice,
		PointsAwarded: ptsAwarded,
		MaxPoints:     maxPts,
	}
}

// TestPValue verifies p = correct_count / n (AC-2: 40/50 → 0.80).
func TestPValue(t *testing.T) {
	var rows []repoitemanalysis.AttemptResponseRow
	for i := 0; i < 50; i++ {
		correct := i < 40
		sp := 0.0
		if correct {
			sp = 100
		}
		rows = append(rows, makeRow(i, 0, "multiple_choice", correct, intPtr(0), boolF(correct), 1, sp))
	}
	items, _ := computeStats(quizID, rows, nowTime)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	got := *items[0].PValue
	if math.Abs(got-0.80) > 0.01 {
		t.Errorf("p-value: got %.4f, want 0.80", got)
	}
}

// TestKR20 validates KR-20 against a hand-calculated 3-item, 5-student example (doubled to 10).
//
// Reference dataset (dichotomous items, 0/1 scoring):
//
//	Student | Q0 | Q1 | Q2 | Raw total
//	  A     |  1 |  1 |  0 |    2
//	  B     |  1 |  0 |  1 |    2
//	  C     |  1 |  1 |  1 |    3
//	  D     |  0 |  1 |  0 |    1
//	  E     |  1 |  0 |  0 |    1
//
// (repeated twice so n=10, statistics unchanged)
//
// p0=0.8, p1=0.6, p2=0.4  →  Σ(p·q) = 0.16+0.24+0.24 = 0.64
// Raw totals [2,2,3,1,1,2,2,3,1,1]  →  mean=1.8, Var_T = 0.56
// KR-20 = (3/2)·(1 − 0.64/0.56) = 1.5·(−0.14286) ≈ −0.2143
func TestKR20(t *testing.T) {
	data := [][3]bool{
		{true, true, false},
		{true, false, true},
		{true, true, true},
		{false, true, false},
		{true, false, false},
	}
	data = append(data, data...) // 10 students

	var rows []repoitemanalysis.AttemptResponseRow
	for i, d := range data {
		sc := 0
		for _, c := range d {
			if c {
				sc++
			}
		}
		sp := float64(sc) / 3.0 * 100 // score_percent
		for qi, correct := range d {
			rows = append(rows, makeRow(i, qi, "multiple_choice", correct, intPtr(0), boolF(correct), 1, sp))
		}
	}

	_, testStat := computeStats(quizID, rows, nowTime)
	if testStat.KR20 == nil {
		t.Fatal("KR20 should be computed for all-dichotomous quiz")
	}
	if testStat.CronbachAlpha != nil {
		t.Fatal("Cronbach alpha should be nil for all-dichotomous quiz")
	}
	const want = -0.2143
	if math.Abs(*testStat.KR20-want) > 0.002 {
		t.Errorf("KR-20: got %.4f, want %.4f (±0.002)", *testStat.KR20, want)
	}
}

// TestCronbachAlpha uses the same dataset as TestKR20 but with one "essay" item,
// triggering the Cronbach α path. For binary items, α == KR-20.
func TestCronbachAlpha(t *testing.T) {
	data := [][3]bool{
		{true, true, false},
		{true, false, true},
		{true, true, true},
		{false, true, false},
		{true, false, false},
	}
	data = append(data, data...)

	var rows []repoitemanalysis.AttemptResponseRow
	for i, d := range data {
		sc := 0
		for _, c := range d {
			if c {
				sc++
			}
		}
		sp := float64(sc) / 3.0 * 100
		for qi, correct := range d {
			qtype := "multiple_choice"
			if qi == 2 {
				qtype = "essay" // forces Cronbach path
			}
			rows = append(rows, makeRow(i, qi, qtype, correct, nil, boolF(correct), 1, sp))
		}
	}

	_, testStat := computeStats(quizID, rows, nowTime)
	if testStat.CronbachAlpha == nil {
		t.Fatal("Cronbach alpha should be computed for mixed quiz")
	}
	if testStat.KR20 != nil {
		t.Fatal("KR-20 should be nil for mixed quiz")
	}
	const want = -0.2143
	if math.Abs(*testStat.CronbachAlpha-want) > 0.002 {
		t.Errorf("Cronbach alpha: got %.4f, want %.4f (±0.002)", *testStat.CronbachAlpha, want)
	}
}

// TestInsufficientData verifies that < MinResponses rows returns empty item stats.
func TestInsufficientData(t *testing.T) {
	var rows []repoitemanalysis.AttemptResponseRow
	for i := 0; i < MinResponses-1; i++ {
		rows = append(rows, makeRow(i, 0, "multiple_choice", true, intPtr(0), 1, 1, 100))
	}
	items, testStat := computeStats(quizID, rows, nowTime)
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
	if testStat.NResponses != MinResponses-1 {
		t.Errorf("n_responses: got %d, want %d", testStat.NResponses, MinResponses-1)
	}
}

// TestFlags checks hard / easy / poor_discriminator thresholds.
func TestFlags(t *testing.T) {
	// p < 0.20 → hard: 1 correct out of 10
	rows := makeUniformRows(10, 1, "multiple_choice")
	items, _ := computeStats(quizID, rows, nowTime)
	requireFlag(t, items, FlagHard)

	// p > 0.90 → easy: 10 correct out of 10 (p=1.0, r_pb undefined → flag=easy)
	rows = makeUniformRows(10, 10, "multiple_choice")
	items, _ = computeStats(quizID, rows, nowTime)
	requireFlag(t, items, FlagEasy)
}

// TestMeanStdDev checks mean_score and std_dev on the test stat.
//
// 10 students: 5 score 100%, 5 score 0% → mean=50, std=50.
func TestMeanStdDev(t *testing.T) {
	var rows []repoitemanalysis.AttemptResponseRow
	for i := 0; i < 10; i++ {
		correct := i < 5
		sp := 0.0
		if correct {
			sp = 100
		}
		rows = append(rows, makeRow(i, 0, "multiple_choice", correct, intPtr(0), boolF(correct), 1, sp))
	}
	_, testStat := computeStats(quizID, rows, nowTime)
	if testStat.MeanScore == nil {
		t.Fatal("mean score should be set")
	}
	if math.Abs(*testStat.MeanScore-50) > 0.01 {
		t.Errorf("mean_score: got %.2f, want 50", *testStat.MeanScore)
	}
	if testStat.StdDev == nil {
		t.Fatal("std_dev should be set")
	}
	if math.Abs(*testStat.StdDev-50) > 0.01 {
		t.Errorf("std_dev: got %.2f, want 50", *testStat.StdDev)
	}
}

// TestDistractorFreqs checks that distractor frequencies sum to 1.0 for MC items.
func TestDistractorFreqs(t *testing.T) {
	choices := []int{0, 1, 2, 3, 0, 1, 2, 3, 0, 1}
	var rows []repoitemanalysis.AttemptResponseRow
	for i, c := range choices {
		ci := c
		sp := 0.0
		if c == 0 {
			sp = 100
		}
		rows = append(rows, makeRow(i, 0, "multiple_choice", c == 0, &ci, boolF(c == 0), 1, sp))
	}
	items, _ := computeStats(quizID, rows, nowTime)
	if len(items) == 0 {
		t.Fatal("expected item stats")
	}
	if items[0].DistractorFreqs == nil {
		t.Fatal("distractor freqs should be set for MC")
	}
	var total float64
	for _, f := range items[0].DistractorFreqs {
		total += f
	}
	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("distractor freqs sum: got %.4f, want 1.0", total)
	}
}

// Helpers

func boolF(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func makeUniformRows(n, nCorrect int, qtype string) []repoitemanalysis.AttemptResponseRow {
	var rows []repoitemanalysis.AttemptResponseRow
	for i := 0; i < n; i++ {
		correct := i < nCorrect
		sp := boolF(correct) * 100
		rows = append(rows, makeRow(i, 0, qtype, correct, intPtr(0), boolF(correct), 1, sp))
	}
	return rows
}

func requireFlag(t *testing.T, items []repoitemanalysis.ItemStatRow, want string) {
	t.Helper()
	if len(items) == 0 {
		t.Fatal("expected at least one item stat")
	}
	if items[0].Flag == nil {
		t.Fatalf("expected flag %q but flag is nil", want)
	}
	if *items[0].Flag != want {
		t.Errorf("flag: got %q, want %q", *items[0].Flag, want)
	}
}
