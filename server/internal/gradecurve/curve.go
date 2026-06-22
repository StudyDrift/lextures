// Package gradecurve implements grade curving and scaling math (plan 3.17).
package gradecurve

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/google/uuid"
)

// Method identifies a curve algorithm.
type Method string

const (
	MethodFlatBonus     Method = "flat_bonus"
	MethodLinearScale   Method = "linear_scale"
	MethodSqrtCurve     Method = "sqrt_curve"
	MethodSetMinimum    Method = "set_minimum"
	MethodCustomMapping Method = "custom_mapping"
)

// Params holds method-specific curve parameters.
type Params struct {
	Bonus      *float64           `json:"bonus,omitempty"`
	TargetMean *float64           `json:"targetMean,omitempty"`
	TargetMax  *float64           `json:"targetMax,omitempty"`
	Minimum    *float64           `json:"minimum,omitempty"`
	Mapping    map[string]float64 `json:"mapping,omitempty"`
}

// ScoreInput is one student's raw score for curve computation.
type ScoreInput struct {
	StudentID uuid.UUID
	RawScore  float64
	Excused   bool
}

// ScoreOutput is the computed curve result for one student.
type ScoreOutput struct {
	StudentID     uuid.UUID `json:"studentId"`
	RawScore      float64   `json:"rawScore"`
	AdjustedScore float64   `json:"adjustedScore"`
	Delta         float64   `json:"delta"`
	Changed       bool      `json:"changed"`
}

// HistogramBucket is one bar in a distribution preview.
type HistogramBucket struct {
	Label string  `json:"label"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

// PreviewSummary aggregates before/after distribution stats.
type PreviewSummary struct {
	EligibleCount  int               `json:"eligibleCount"`
	MeanBefore     *float64          `json:"meanBefore"`
	MeanAfter      *float64          `json:"meanAfter"`
	MedianBefore   *float64          `json:"medianBefore"`
	MedianAfter    *float64          `json:"medianAfter"`
	HistogramBefore []HistogramBucket `json:"histogramBefore"`
	HistogramAfter  []HistogramBucket `json:"histogramAfter"`
	Results        []ScoreOutput     `json:"results"`
}

// Options configures curve application.
type Options struct {
	MaxPoints     float64
	AllowAboveMax bool
	Method        Method
	Params        Params
}

// ParseParams unmarshals params JSON into Params.
func ParseParams(raw json.RawMessage) (Params, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return Params{}, nil
	}
	var p Params
	if err := json.Unmarshal(raw, &p); err != nil {
		return Params{}, err
	}
	return p, nil
}

// ValidateMethod checks that method and params are compatible.
func ValidateMethod(method Method, params Params) error {
	switch method {
	case MethodFlatBonus:
		if params.Bonus == nil {
			return fmt.Errorf("flat_bonus requires bonus")
		}
	case MethodLinearScale:
		if params.TargetMean == nil && params.TargetMax == nil {
			return fmt.Errorf("linear_scale requires targetMean or targetMax")
		}
		if params.TargetMean != nil && params.TargetMax != nil {
			return fmt.Errorf("linear_scale accepts only one of targetMean or targetMax")
		}
	case MethodSqrtCurve:
		return nil
	case MethodSetMinimum:
		if params.Minimum == nil {
			return fmt.Errorf("set_minimum requires minimum")
		}
	case MethodCustomMapping:
		if len(params.Mapping) == 0 {
			return fmt.Errorf("custom_mapping requires mapping")
		}
	default:
		return fmt.Errorf("unknown curve method %q", method)
	}
	return nil
}

// Preview computes adjusted scores and distribution summary without persisting.
func Preview(scores []ScoreInput, opts Options) (PreviewSummary, error) {
	if err := ValidateMethod(opts.Method, opts.Params); err != nil {
		return PreviewSummary{}, err
	}
	eligible := filterEligible(scores)
	adjustedByStudent := applyMethod(eligible, opts)
	out := PreviewSummary{
		EligibleCount: len(eligible),
		Results:       make([]ScoreOutput, 0, len(scores)),
	}
	beforeVals := make([]float64, 0, len(eligible))
	afterVals := make([]float64, 0, len(eligible))
	for _, s := range eligible {
		raw := s.RawScore
		adj, ok := adjustedByStudent[s.StudentID]
		if !ok {
			adj = raw
		}
		beforeVals = append(beforeVals, raw)
		afterVals = append(afterVals, adj)
	}
	for _, s := range scores {
		if s.Excused {
			continue
		}
		adj, ok := adjustedByStudent[s.StudentID]
		if !ok {
			continue
		}
		delta := round2(adj - s.RawScore)
		out.Results = append(out.Results, ScoreOutput{
			StudentID:     s.StudentID,
			RawScore:      round2(s.RawScore),
			AdjustedScore: round2(adj),
			Delta:         delta,
			Changed:       math.Abs(delta) > 1e-9,
		})
	}
	sort.Slice(out.Results, func(i, j int) bool {
		return out.Results[i].StudentID.String() < out.Results[j].StudentID.String()
	})
	out.MeanBefore = meanPtr(beforeVals)
	out.MeanAfter = meanPtr(afterVals)
	out.MedianBefore = medianPtr(beforeVals)
	out.MedianAfter = medianPtr(afterVals)
	out.HistogramBefore = histogram(beforeVals, opts.MaxPoints)
	out.HistogramAfter = histogram(afterVals, opts.MaxPoints)
	return out, nil
}

func filterEligible(scores []ScoreInput) []ScoreInput {
	out := make([]ScoreInput, 0, len(scores))
	for _, s := range scores {
		if s.Excused {
			continue
		}
		if math.IsNaN(s.RawScore) || math.IsInf(s.RawScore, 0) || s.RawScore < 0 {
			continue
		}
		out = append(out, s)
	}
	return out
}

func applyMethod(eligible []ScoreInput, opts Options) map[uuid.UUID]float64 {
	out := make(map[uuid.UUID]float64, len(eligible))
	if len(eligible) == 0 {
		return out
	}
	switch opts.Method {
	case MethodFlatBonus:
		bonus := *opts.Params.Bonus
		for _, s := range eligible {
			out[s.StudentID] = capScore(s.RawScore+bonus, opts)
		}
	case MethodLinearScale:
		if opts.Params.TargetMean != nil {
			target := *opts.Params.TargetMean
			vals := scoresOnly(eligible)
			m := mean(vals)
			shift := target - m
			for _, s := range eligible {
				out[s.StudentID] = capScore(s.RawScore+shift, opts)
			}
		} else {
			targetMax := *opts.Params.TargetMax
			curMax := maxScore(eligible)
			if curMax <= 0 {
				for _, s := range eligible {
					out[s.StudentID] = capScore(s.RawScore, opts)
				}
				return out
			}
			scale := targetMax / curMax
			for _, s := range eligible {
				out[s.StudentID] = capScore(s.RawScore*scale, opts)
			}
		}
	case MethodSqrtCurve:
		maxPts := opts.MaxPoints
		if maxPts <= 0 {
			for _, s := range eligible {
				out[s.StudentID] = capScore(s.RawScore, opts)
			}
			return out
		}
		for _, s := range eligible {
			ratio := s.RawScore / maxPts
			if ratio < 0 {
				ratio = 0
			}
			out[s.StudentID] = capScore(math.Sqrt(ratio)*maxPts, opts)
		}
	case MethodSetMinimum:
		floor := *opts.Params.Minimum
		for _, s := range eligible {
			v := s.RawScore
			if v < floor {
				v = floor
			}
			out[s.StudentID] = capScore(v, opts)
		}
	case MethodCustomMapping:
		for _, s := range eligible {
			key := formatMappingKey(s.RawScore)
			if adj, ok := opts.Params.Mapping[key]; ok {
				out[s.StudentID] = capScore(adj, opts)
			} else {
				out[s.StudentID] = capScore(s.RawScore, opts)
			}
		}
	default:
		for _, s := range eligible {
			out[s.StudentID] = capScore(s.RawScore, opts)
		}
	}
	return out
}

func capScore(v float64, opts Options) float64 {
	if v < 0 {
		v = 0
	}
	if !opts.AllowAboveMax && opts.MaxPoints > 0 && v > opts.MaxPoints {
		v = opts.MaxPoints
	}
	return round2(v)
}

func maxScore(in []ScoreInput) float64 {
	max := 0.0
	for _, s := range in {
		if s.RawScore > max {
			max = s.RawScore
		}
	}
	return max
}

func scoresOnly(in []ScoreInput) []float64 {
	out := make([]float64, len(in))
	for i, s := range in {
		out[i] = s.RawScore
	}
	return out
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func meanPtr(vals []float64) *float64 {
	if len(vals) == 0 {
		return nil
	}
	m := round2(mean(vals))
	return &m
}

func medianPtr(vals []float64) *float64 {
	if len(vals) == 0 {
		return nil
	}
	cp := append([]float64(nil), vals...)
	sort.Float64s(cp)
	n := len(cp)
	var med float64
	if n%2 == 1 {
		med = cp[n/2]
	} else {
		med = (cp[n/2-1] + cp[n/2]) / 2
	}
	med = round2(med)
	return &med
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func formatMappingKey(v float64) string {
	return strconv.FormatFloat(round2(v), 'f', -1, 64)
}

func histogram(vals []float64, maxPts float64) []HistogramBucket {
	if maxPts <= 0 {
		maxPts = 100
	}
	const buckets = 10
	width := maxPts / buckets
	if width <= 0 {
		width = 1
	}
	out := make([]HistogramBucket, buckets)
	for i := range out {
		min := float64(i) * width
		max := float64(i+1) * width
		if i == buckets-1 {
			max = maxPts
		}
		out[i] = HistogramBucket{
			Label: fmt.Sprintf("%.0f–%.0f", min, max),
			Min:   min,
			Max:   max,
		}
	}
	for _, v := range vals {
		idx := int(v / width)
		if idx >= buckets {
			idx = buckets - 1
		}
		if idx < 0 {
			idx = 0
		}
		out[idx].Count++
	}
	return out
}

// AuditReasonJSON builds a JSON reason string for grade audit events.
func AuditReasonJSON(method Method, params Params, curveID uuid.UUID) string {
	payload := map[string]any{
		"curveId": curveID.String(),
		"method":  string(method),
		"params":  params,
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
