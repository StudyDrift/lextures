package analytics

import (
	"math"
	"sort"
)

// MinRespondentsFloor is the hard floor for peer-facing aggregate suppression (CT.21).
const MinRespondentsFloor = 3

// CountedOption is one option's anonymised count in a distribution.
type CountedOption struct {
	OptionID string  `json:"optionId"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent,omitempty"`
}

// AggregateWithSuppressionResult is a peer-safe option distribution.
type AggregateWithSuppressionResult struct {
	Suppressed bool            `json:"suppressed"`
	Reason     string          `json:"reason,omitempty"`
	Learners   int             `json:"learners"`
	Options    []CountedOption `json:"options,omitempty"`
}

// ClampMinRespondents applies the CT.21 floor (3) and default (DefaultSmallN).
func ClampMinRespondents(n int) int {
	if n <= 0 {
		return DefaultSmallN
	}
	if n < MinRespondentsFloor {
		return MinRespondentsFloor
	}
	return n
}

// AggregateWithSuppression builds an anonymised option distribution, withholding
// counts when learners < minRespondents (server-side small-n suppression).
func AggregateWithSuppression(optionCounts map[string]int, learners, minRespondents int, showPercentages bool) AggregateWithSuppressionResult {
	minRespondents = ClampMinRespondents(minRespondents)
	out := AggregateWithSuppressionResult{Learners: learners}
	if learners < minRespondents {
		out.Suppressed = true
		out.Reason = "small_n"
		return out
	}
	ids := make([]string, 0, len(optionCounts))
	for id := range optionCounts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out.Options = make([]CountedOption, 0, len(ids))
	for _, id := range ids {
		c := optionCounts[id]
		opt := CountedOption{OptionID: id, Count: c}
		if showPercentages && learners > 0 {
			opt.Percent = math.Round((float64(c)/float64(learners))*1000) / 10
		}
		out.Options = append(out.Options, opt)
	}
	return out
}
