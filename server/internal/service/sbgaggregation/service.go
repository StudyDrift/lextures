// Package sbgaggregation computes final mastery levels per student per standard
// using configurable aggregation strategies (plan 13.5).
package sbgaggregation

import (
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/sbgreport"
)

// Method is the aggregation strategy name.
type Method string

const (
	// MostRecent returns the score from the most recent assessment.
	MostRecent Method = "most_recent"
	// Highest returns the highest score across all assessments.
	Highest Method = "highest"
	// Mode returns the most frequently occurring score (ties broken by highest).
	Mode Method = "mode"
	// Trend returns the average of the most recent 3 assessments.
	Trend Method = "trend"
)

// AggregatedScore is the computed final mastery level for one student+standard pair.
type AggregatedScore struct {
	StudentID  uuid.UUID
	StandardID uuid.UUID
	ScoreValue int
	Method     Method
}

// Aggregate computes the final mastery level for each student+standard pair in
// scores, using the given method. Scores must be ordered by assessed_at ascending
// (oldest first) — the repo always returns them that way.
func Aggregate(scores []sbgreport.MasteryScore, method Method) []AggregatedScore {
	// Group by (student, standard).
	type key struct{ student, standard uuid.UUID }
	grouped := map[key][]int{}
	for _, s := range scores {
		k := key{s.StudentID, s.StandardID}
		grouped[k] = append(grouped[k], s.ScoreValue)
	}

	var out []AggregatedScore
	for k, vals := range grouped {
		out = append(out, AggregatedScore{
			StudentID:  k.student,
			StandardID: k.standard,
			ScoreValue: aggregate(vals, method),
			Method:     method,
		})
	}
	return out
}

func aggregate(vals []int, method Method) int {
	if len(vals) == 0 {
		return 0
	}
	switch method {
	case MostRecent:
		return vals[len(vals)-1]
	case Highest:
		max := vals[0]
		for _, v := range vals[1:] {
			if v > max {
				max = v
			}
		}
		return max
	case Mode:
		return mode(vals)
	case Trend:
		return trend(vals)
	default:
		return vals[len(vals)-1]
	}
}

func mode(vals []int) int {
	freq := map[int]int{}
	for _, v := range vals {
		freq[v]++
	}
	bestVal, bestFreq := 0, 0
	for v, f := range freq {
		if f > bestFreq || (f == bestFreq && v > bestVal) {
			bestVal, bestFreq = v, f
		}
	}
	return bestVal
}

func trend(vals []int) int {
	// Average of the most recent 3 (or fewer if < 3 assessments).
	n := 3
	if len(vals) < n {
		n = len(vals)
	}
	recent := vals[len(vals)-n:]
	sum := 0
	for _, v := range recent {
		sum += v
	}
	// Integer rounding: round to nearest.
	avg := (sum*10/n + 5) / 10
	return avg
}

// AggregateForReport is a convenience wrapper used by the SBG report card handler.
// It returns a map[studentID][standardID]scoreValue.
func AggregateForReport(scores []sbgreport.MasteryScore, method Method) map[uuid.UUID]map[uuid.UUID]int {
	agg := Aggregate(scores, method)
	out := make(map[uuid.UUID]map[uuid.UUID]int)
	for _, a := range agg {
		if out[a.StudentID] == nil {
			out[a.StudentID] = make(map[uuid.UUID]int)
		}
		out[a.StudentID][a.StandardID] = a.ScoreValue
	}
	return out
}

// ParseMethod converts a string to Method, defaulting to MostRecent for unknown values.
func ParseMethod(s string) Method {
	switch Method(s) {
	case MostRecent, Highest, Mode, Trend:
		return Method(s)
	default:
		return MostRecent
	}
}
