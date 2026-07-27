package analytics

import (
	"math"
	"sort"
	"strconv"
)

// StaffRoles are enrollment roles excluded from learner aggregates (FR-5).
var StaffRoles = map[string]struct{}{
	"teacher": {}, "instructor": {}, "ta": {}, "teaching_assistant": {},
	"designer": {}, "observer": {}, "admin": {},
}

// IsLearnerRole reports whether the enrollment role counts toward aggregates.
func IsLearnerRole(role string) bool {
	if role == "" {
		return false
	}
	_, staff := StaffRoles[role]
	return !staff
}

// ScoreBucket is one histogram bin.
type ScoreBucket struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// FacetValueCount is one facet value with its count.
type FacetValueCount struct {
	Value   string `json:"value"`
	Count   int    `json:"count"`
	Correct *bool  `json:"correct,omitempty"`
}

// FacetDistribution is aggregated facet counts for one key.
type FacetDistribution struct {
	Key    string            `json:"key"`
	Label  string            `json:"label"`
	Values []FacetValueCount `json:"values"`
}

// NeedsAttentionReason classifies why a learner needs attention.
type NeedsAttentionReason string

const (
	AttentionNotStarted NeedsAttentionReason = "not_started"
	AttentionStuck      NeedsAttentionReason = "stuck"
	AttentionLowScore   NeedsAttentionReason = "low_score"
)

// NeedsAttentionRow is one learner flagged for instructor follow-up.
type NeedsAttentionRow struct {
	EnrollmentID string               `json:"enrollmentId"`
	DisplayName  string               `json:"displayName"`
	Reason       NeedsAttentionReason `json:"reason"`
}

// SummaryRow is a persisted summary plus display name for roster surfaces.
type SummaryRow struct {
	EnrollmentID string
	DisplayName  string
	Role         string
	Engaged      bool
	Completed    bool
	ScorePct     *float64
	DurationMs   *int
	Facets       map[string]any
	Status       string // optional raw status when available
}

// InstanceAggregate is the instructor analytics payload for one instance (FR-3).
type InstanceAggregate struct {
	Learners         int                 `json:"learners"`
	Engaged          int                 `json:"engaged"`
	Completed        int                 `json:"completed"`
	Suppressed       bool                `json:"suppressed"`
	ScoreMean        *float64            `json:"scoreMean,omitempty"`
	ScoreMedian      *float64            `json:"scoreMedian,omitempty"`
	ScoreDistribution []ScoreBucket      `json:"scoreDistribution,omitempty"`
	MedianDurationMs *int                `json:"medianDurationMs,omitempty"`
	Facets           []FacetDistribution `json:"facets"`
	NeedsAttention   []NeedsAttentionRow `json:"needsAttention"`
}

// AggregateInstance computes instance-level aggregates from summary rows.
// smallN: when > 0 and learners < smallN, numeric aggregates are suppressed (FR-6)
// except needsAttention which is always identified for the instructor roster.
func AggregateInstance(rows []SummaryRow, toolID string, smallN int, suppress bool) InstanceAggregate {
	if smallN <= 0 {
		smallN = DefaultSmallN
	}
	learners := make([]SummaryRow, 0, len(rows))
	for _, r := range rows {
		if IsLearnerRole(r.Role) {
			learners = append(learners, r)
		}
	}
	out := InstanceAggregate{
		Learners:       len(learners),
		Facets:         []FacetDistribution{},
		NeedsAttention: []NeedsAttentionRow{},
	}
	for _, r := range learners {
		if r.Engaged {
			out.Engaged++
		}
		if r.Completed {
			out.Completed++
		}
		out.NeedsAttention = append(out.NeedsAttention, classifyAttention(r)...)
	}
	out.NeedsAttention = dedupeAttention(out.NeedsAttention)

	if suppress && out.Learners > 0 && out.Learners < smallN {
		out.Suppressed = true
		return out
	}

	scores := make([]float64, 0, len(learners))
	durs := make([]int, 0, len(learners))
	for _, r := range learners {
		if r.ScorePct != nil {
			scores = append(scores, *r.ScorePct)
		}
		if r.DurationMs != nil {
			durs = append(durs, *r.DurationMs)
		}
	}
	if len(scores) > 0 {
		mean := meanFloat(scores)
		med := medianFloat(scores)
		out.ScoreMean = &mean
		out.ScoreMedian = &med
		out.ScoreDistribution = scoreHistogram(scores)
	}
	if len(durs) > 0 {
		m := medianInt(durs)
		out.MedianDurationMs = &m
	}
	out.Facets = aggregateFacets(learners, toolID)
	return out
}

func classifyAttention(r SummaryRow) []NeedsAttentionRow {
	var out []NeedsAttentionRow
	if !r.Engaged || r.Status == "not_started" {
		out = append(out, NeedsAttentionRow{
			EnrollmentID: r.EnrollmentID,
			DisplayName:  r.DisplayName,
			Reason:       AttentionNotStarted,
		})
		return out
	}
	if r.Engaged && !r.Completed {
		out = append(out, NeedsAttentionRow{
			EnrollmentID: r.EnrollmentID,
			DisplayName:  r.DisplayName,
			Reason:       AttentionStuck,
		})
	}
	if r.ScorePct != nil && *r.ScorePct < 50 && r.Completed {
		out = append(out, NeedsAttentionRow{
			EnrollmentID: r.EnrollmentID,
			DisplayName:  r.DisplayName,
			Reason:       AttentionLowScore,
		})
	}
	return out
}

func dedupeAttention(in []NeedsAttentionRow) []NeedsAttentionRow {
	seen := map[string]NeedsAttentionRow{}
	order := []string{}
	priority := map[NeedsAttentionReason]int{
		AttentionNotStarted: 3,
		AttentionStuck:      2,
		AttentionLowScore:   1,
	}
	for _, row := range in {
		prev, ok := seen[row.EnrollmentID]
		if !ok || priority[row.Reason] > priority[prev.Reason] {
			if !ok {
				order = append(order, row.EnrollmentID)
			}
			seen[row.EnrollmentID] = row
		}
	}
	out := make([]NeedsAttentionRow, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out
}

func aggregateFacets(rows []SummaryRow, toolID string) []FacetDistribution {
	schemas := FacetsForTool(toolID)
	if len(schemas) == 0 {
		return []FacetDistribution{}
	}
	out := make([]FacetDistribution, 0, len(schemas))
	for _, schema := range schemas {
		counts := map[string]int{}
		for _, r := range rows {
			if r.Facets == nil {
				continue
			}
			v, ok := r.Facets[schema.Key]
			if !ok || v == nil {
				continue
			}
			key := facetValueKey(v)
			counts[key]++
		}
		vals := make([]FacetValueCount, 0, len(counts))
		for k, c := range counts {
			fv := FacetValueCount{Value: k, Count: c}
			if schema.Key == "correct" {
				b := k == "true"
				fv.Correct = &b
			}
			vals = append(vals, fv)
		}
		sort.Slice(vals, func(i, j int) bool {
			if vals[i].Count != vals[j].Count {
				return vals[i].Count > vals[j].Count
			}
			return vals[i].Value < vals[j].Value
		})
		out = append(out, FacetDistribution{Key: schema.Key, Label: schema.Label, Values: vals})
	}
	return out
}

func facetValueKey(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case string:
		return t
	default:
		b, _ := jsonMarshal(v)
		return string(b)
	}
}

func jsonMarshal(v any) ([]byte, error) {
	return jsonMarshalImpl(v)
}

func meanFloat(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return math.Round((s/float64(len(xs)))*100) / 100
}

func medianFloat(xs []float64) float64 {
	cp := append([]float64{}, xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return math.Round(((cp[n/2-1]+cp[n/2])/2)*100) / 100
}

func medianInt(xs []int) int {
	cp := append([]int{}, xs...)
	sort.Ints(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func scoreHistogram(scores []float64) []ScoreBucket {
	buckets := []struct {
		label string
		lo, hi float64
	}{
		{"0-19", 0, 20},
		{"20-39", 20, 40},
		{"40-59", 40, 60},
		{"60-79", 60, 80},
		{"80-100", 80, 101},
	}
	out := make([]ScoreBucket, len(buckets))
	for i, b := range buckets {
		out[i] = ScoreBucket{Bucket: b.label}
	}
	for _, s := range scores {
		for i, b := range buckets {
			if s >= b.lo && s < b.hi {
				out[i].Count++
				break
			}
		}
	}
	return out
}
