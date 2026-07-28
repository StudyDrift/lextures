package class_pulse

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/contenttools/analytics"
)

// VoteRow is one enrollment's vote contribution for aggregation.
type VoteRow struct {
	EnrollmentID uuid.UUID
	Role         string
	SectionID    *uuid.UUID
	State        State
}

// RoundAggregate is the anonymised distribution for one voting round.
type RoundAggregate struct {
	Round      int                              `json:"round"`
	Suppressed bool                             `json:"suppressed"`
	Reason     string                           `json:"reason,omitempty"`
	Learners   int                              `json:"learners"`
	Options    []analytics.CountedOption        `json:"options,omitempty"`
}

// ShiftCell counts how many learners moved from one option to another between rounds.
type ShiftCell struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Count int    `json:"count"`
}

// BuildRoundAggregate counts learner votes for a round with role/section filters
// and small-n suppression (FR-3, FR-6).
func BuildRoundAggregate(rows []VoteRow, round int, cfg Config, sectionFilter *uuid.UUID) RoundAggregate {
	counts := map[string]int{}
	learners := 0
	for _, row := range rows {
		if !analytics.IsLearnerRole(row.Role) {
			continue
		}
		if sectionFilter != nil {
			if row.SectionID == nil || *row.SectionID != *sectionFilter {
				continue
			}
		}
		v := row.State.VoteForRound(round)
		if v == nil {
			continue
		}
		learners++
		counts[v.OptionID]++
	}
	res := analytics.AggregateWithSuppression(counts, learners, cfg.MinRespondents, cfg.ShowPercentages)
	return RoundAggregate{
		Round:      round,
		Suppressed: res.Suppressed,
		Reason:     res.Reason,
		Learners:   res.Learners,
		Options:    res.Options,
	}
}

// BuildShift computes first→second vote movement (instructor analytics).
func BuildShift(rows []VoteRow, sectionFilter *uuid.UUID) []ShiftCell {
	type key struct{ from, to string }
	counts := map[key]int{}
	for _, row := range rows {
		if !analytics.IsLearnerRole(row.Role) {
			continue
		}
		if sectionFilter != nil {
			if row.SectionID == nil || *row.SectionID != *sectionFilter {
				continue
			}
		}
		v1 := row.State.VoteForRound(1)
		v2 := row.State.VoteForRound(2)
		if v1 == nil || v2 == nil {
			continue
		}
		counts[key{v1.OptionID, v2.OptionID}]++
	}
	out := make([]ShiftCell, 0, len(counts))
	for k, n := range counts {
		out = append(out, ShiftCell{From: k.from, To: k.to, Count: n})
	}
	return out
}

// RowsFromStateBlobs adapts raw state JSON (+ metadata) into VoteRows.
func RowsFromStateBlobs(enrollmentID uuid.UUID, role string, sectionID *uuid.UUID, stateJSON json.RawMessage) VoteRow {
	return VoteRow{
		EnrollmentID: enrollmentID,
		Role:         role,
		SectionID:    sectionID,
		State:        ParseState(stateJSON),
	}
}
