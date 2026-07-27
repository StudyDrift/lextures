package predict_reveal

import (
	"encoding/json"

	"github.com/lextures/lextures/server/internal/service/contenttools/analytics"
)

// PeerResults is the anonymised peer distribution returned after commit.
type PeerResults struct {
	Suppressed        bool               `json:"suppressed"`
	Reason            string             `json:"reason,omitempty"`
	Learners          int                `json:"learners"`
	Outcomes          []PeerOutcomeCount `json:"outcomes,omitempty"`
	ConfidenceBuckets []PeerBucketCount  `json:"confidenceBuckets,omitempty"`
}

// PeerOutcomeCount is one choice outcome's peer count.
type PeerOutcomeCount struct {
	OutcomeID string `json:"outcomeId"`
	Count     int    `json:"count"`
}

// PeerBucketCount is one confidence bucket's peer count.
type PeerBucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// BuildPeerResults aggregates committed states into an anonymous distribution.
// When respondents < DefaultSmallN, results are suppressed (FR-10 / AC-6).
func BuildPeerResults(stateBlobs []json.RawMessage, mode Mode) PeerResults {
	outcomeCounts := map[string]int{}
	bucketCounts := map[string]int{}
	learners := 0
	for _, raw := range stateBlobs {
		st := ParseState(raw)
		if !st.IsCommitted() {
			continue
		}
		learners++
		if mode == ModeChoice && st.Prediction != nil && st.Prediction.OutcomeID != "" {
			outcomeCounts[st.Prediction.OutcomeID]++
		}
		if st.ConfidenceBucket != "" {
			bucketCounts[st.ConfidenceBucket]++
		}
	}
	out := PeerResults{Learners: learners}
	if learners < analytics.DefaultSmallN {
		out.Suppressed = true
		out.Reason = "small_n"
		return out
	}
	for id, n := range outcomeCounts {
		out.Outcomes = append(out.Outcomes, PeerOutcomeCount{OutcomeID: id, Count: n})
	}
	for b, n := range bucketCounts {
		out.ConfidenceBuckets = append(out.ConfidenceBuckets, PeerBucketCount{Bucket: b, Count: n})
	}
	return out
}
