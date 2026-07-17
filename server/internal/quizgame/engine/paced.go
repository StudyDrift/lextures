package engine

import (
	"math/rand"
	"time"
)

// ShuffleQuestionOrder returns a permutation of [0..n) for student-paced/homework.
func ShuffleQuestionOrder(n int, rng *rand.Rand) []int {
	order := make([]int, n)
	for i := 0; i < n; i++ {
		order[i] = i
	}
	if n <= 1 {
		return order
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	rng.Shuffle(n, func(i, j int) { order[i], order[j] = order[j], order[i] })
	return order
}

// SequentialQuestionOrder returns [0..n).
func SequentialQuestionOrder(n int) []int {
	order := make([]int, n)
	for i := 0; i < n; i++ {
		order[i] = i
	}
	return order
}

// ResolveQuestionIndex maps a player's progress position to the kit question index.
func ResolveQuestionIndex(order []int, progressIndex int) (kitIndex int, ok bool) {
	if progressIndex < 0 || progressIndex >= len(order) {
		return -1, false
	}
	return order[progressIndex], true
}

// PacedDeadline computes the per-question deadline for a player.
func PacedDeadline(openedAt time.Time, timeLimitSeconds int, perQuestionTimers bool) *time.Time {
	if !perQuestionTimers || timeLimitSeconds <= 0 {
		return nil
	}
	d := openedAt.Add(time.Duration(timeLimitSeconds) * time.Second)
	return &d
}

// TimeBudgetExpired reports whether the overall paced budget has elapsed (AC-4).
func TimeBudgetExpired(endsAt *time.Time, now time.Time) bool {
	if endsAt == nil {
		return false
	}
	return !now.Before(endsAt.UTC())
}

// ProgressBucket counts how many players have reached at least questionIndex (host aggregate).
type ProgressBucket struct {
	QuestionIndex int `json:"questionIndex"`
	Reached       int `json:"reached"` // players with current_index >= this OR finished
	Finished      int `json:"finished"`
}

// AggregatePacedProgress builds host-facing progress (AC-3).
// currentIndices[i] is player current_index; finished[i] marks completed players.
func AggregatePacedProgress(questionCount int, currentIndices []int, finished []bool) []ProgressBucket {
	if questionCount < 0 {
		questionCount = 0
	}
	out := make([]ProgressBucket, questionCount)
	for i := 0; i < questionCount; i++ {
		out[i].QuestionIndex = i
	}
	for i, idx := range currentIndices {
		fin := i < len(finished) && finished[i]
		if fin {
			for j := 0; j < questionCount; j++ {
				out[j].Reached++
			}
			if questionCount > 0 {
				out[questionCount-1].Finished++
			}
			continue
		}
		for j := 0; j <= idx && j < questionCount; j++ {
			out[j].Reached++
		}
	}
	return out
}
