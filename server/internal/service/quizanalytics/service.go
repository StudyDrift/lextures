// Package quizanalytics builds instructor-facing quiz performance reports.
package quizanalytics

import (
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
)

const bucketCount = 10

// ScoreBucket is one histogram bin for overall quiz scores (0–100%).
type ScoreBucket struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Count int    `json:"count"`
}

// QuestionStat summarizes performance on one quiz question.
type QuestionStat struct {
	QuestionIndex int     `json:"questionIndex"`
	QuestionText  string  `json:"questionText"`
	NResponses    int     `json:"nResponses"`
	PctCorrect    float64 `json:"pctCorrect"`
}

// AttemptFocusStat summarizes focus-loss events for one submitted attempt.
type AttemptFocusStat struct {
	AttemptID             string `json:"attemptId"`
	AttemptNumber         int32  `json:"attemptNumber"`
	EventCount            int64  `json:"eventCount"`
	AcademicIntegrityFlag bool   `json:"academicIntegrityFlag"`
}

// Report is the analytics payload for a quiz.
type Report struct {
	QuizID        uuid.UUID          `json:"quizId"`
	NAttempts     int                `json:"nAttempts"`
	MeanScore     *float64           `json:"meanScore"`
	ScoreBuckets  []ScoreBucket      `json:"scoreBuckets"`
	QuestionStats []QuestionStat     `json:"questionStats"`
	FocusAttempts []AttemptFocusStat `json:"focusAttempts"`
}

// BuildReport aggregates submitted attempt data into score and question distributions.
func BuildReport(quizID uuid.UUID, rows []repoitemanalysis.AttemptResponseRow) Report {
	buckets := defaultScoreBuckets()
	report := Report{
		QuizID:       quizID,
		ScoreBuckets: buckets,
	}

	type attemptAgg struct {
		scorePct  *float64
		responses map[int]repoitemanalysis.AttemptResponseRow
	}

	attempts := map[uuid.UUID]*attemptAgg{}
	for _, r := range rows {
		agg, ok := attempts[r.AttemptID]
		if !ok {
			agg = &attemptAgg{responses: map[int]repoitemanalysis.AttemptResponseRow{}}
			attempts[r.AttemptID] = agg
		}
		if agg.scorePct == nil && r.ScorePercent != nil {
			agg.scorePct = r.ScorePercent
		}
		agg.responses[r.QuestionIndex] = r
	}

	report.NAttempts = len(attempts)
	if report.NAttempts == 0 {
		return report
	}

	var scoreSum float64
	var scoredAttempts int
	for _, agg := range attempts {
		if agg.scorePct == nil {
			continue
		}
		scoredAttempts++
		scoreSum += *agg.scorePct
		bucketIdx := scoreBucketIndex(*agg.scorePct)
		if bucketIdx >= 0 && bucketIdx < len(buckets) {
			buckets[bucketIdx].Count++
		}
	}
	if scoredAttempts > 0 {
		mean := scoreSum / float64(scoredAttempts)
		report.MeanScore = &mean
	}

	type questionAgg struct {
		text     string
		sumFrac  float64
		count    int
	}
	questions := map[int]*questionAgg{}
	for _, agg := range attempts {
		for qi, r := range agg.responses {
			q, ok := questions[qi]
			if !ok {
				q = &questionAgg{}
				questions[qi] = q
			}
			if r.PromptText != nil && q.text == "" {
				q.text = *r.PromptText
			}
			q.sumFrac += itemPerformanceFraction(r)
			q.count++
		}
	}

	indices := make([]int, 0, len(questions))
	for qi := range questions {
		indices = append(indices, qi)
	}
	sort.Ints(indices)

	report.QuestionStats = make([]QuestionStat, 0, len(indices))
	for _, qi := range indices {
		q := questions[qi]
		if q.count == 0 {
			continue
		}
		report.QuestionStats = append(report.QuestionStats, QuestionStat{
			QuestionIndex: qi,
			QuestionText:  q.text,
			NResponses:    q.count,
			PctCorrect:    round1(q.sumFrac / float64(q.count) * 100),
		})
	}

	return report
}

func defaultScoreBuckets() []ScoreBucket {
	buckets := make([]ScoreBucket, bucketCount)
	for i := range buckets {
		min := i * 10
		max := min + 9
		if i == bucketCount-1 {
			max = 100
		}
		buckets[i] = ScoreBucket{
			Label: bucketLabel(min, max),
			Min:   min,
			Max:   max,
		}
	}
	return buckets
}

func bucketLabel(min, max int) string {
	if min == max {
		return fmt.Sprintf("%d%%", min)
	}
	return fmt.Sprintf("%d–%d%%", min, max)
}

func scoreBucketIndex(scorePct float64) int {
	if scorePct < 0 {
		return 0
	}
	if scorePct >= 100 {
		return bucketCount - 1
	}
	return int(math.Floor(scorePct / 10))
}

func itemPerformanceFraction(r repoitemanalysis.AttemptResponseRow) float64 {
	if r.MaxPoints > 0 {
		frac := r.PointsAwarded / r.MaxPoints
		if frac < 0 {
			return 0
		}
		if frac > 1 {
			return 1
		}
		return frac
	}
	if r.IsCorrect != nil && *r.IsCorrect {
		return 1
	}
	return 0
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
