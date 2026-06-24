package gradingagent

import (
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

// AggregatorMode selects how inbound grades are combined.
type AggregatorMode string

const (
	AggregatorModeSum          AggregatorMode = "sum"
	AggregatorModeWeightedSum  AggregatorMode = "weightedSum"
	AggregatorModeAverage      AggregatorMode = "average"
	AggregatorModeMin          AggregatorMode = "min"
	AggregatorModeMax          AggregatorMode = "max"
	AggregatorModeRubricMerge  AggregatorMode = "rubricMerge"
)

// AggregatorConfidenceMode selects how inbound confidences are combined.
type AggregatorConfidenceMode string

const (
	AggregatorConfidenceMin       AggregatorConfidenceMode = "min"
	AggregatorConfidenceMean      AggregatorConfidenceMode = "mean"
	AggregatorConfidenceWeighted  AggregatorConfidenceMode = "weighted"
)

// AggregatorOnMissing selects behavior when an inbound grade is absent.
type AggregatorOnMissing string

const (
	AggregatorOnMissingTreatAsZero        AggregatorOnMissing = "treatAsZero"
	AggregatorOnMissingSkipAndRenormalize AggregatorOnMissing = "skipAndRenormalize"
	AggregatorOnMissingFailItem           AggregatorOnMissing = "failItem"
)

// AggregatorInput is one inbound grade wired into a Score Aggregator.
type AggregatorInput struct {
	SourceID string
	Label    string
	Grade    *GradeOutput
	Weight   float64
	Missing  bool
}

// AggregatorConfig holds Score Aggregator node settings.
type AggregatorConfig struct {
	Mode          AggregatorMode
	Weights       map[string]float64
	Confidence    AggregatorConfidenceMode
	OnMissing     AggregatorOnMissing
	MergeComments bool
	CommentSep    string
}

// CombineGrades folds inbound grades into one output deterministically.
func CombineGrades(
	inputs []AggregatorInput,
	cfg AggregatorConfig,
	maxPoints float64,
	rubric *assignmentrubric.RubricDefinition,
) (GradeOutput, []string, error) {
	if len(inputs) == 0 {
		return GradeOutput{}, nil, fmt.Errorf("aggregator requires at least one grade input")
	}

	present, missingCount, err := resolveAggregatorInputs(inputs, cfg.OnMissing)
	if err != nil {
		return GradeOutput{}, nil, err
	}
	if len(present) == 0 {
		return GradeOutput{}, nil, fmt.Errorf("aggregator has no usable grade inputs")
	}

	logs := make([]string, 0, len(inputs)+2)
	for _, in := range inputs {
		if in.Missing {
			logs = append(logs, fmt.Sprintf("  • %s: missing (policy %s)", in.Label, cfg.OnMissing))
			continue
		}
		pts := 0.0
		conf := 0.0
		if in.Grade != nil {
			pts = in.Grade.TotalPoints
			conf = in.Grade.Confidence
		}
		if cfg.Mode == AggregatorModeWeightedSum {
			logs = append(logs, fmt.Sprintf("  • %s: %.2f (weight %.3f, confidence %.0f%%)", in.Label, pts, in.Weight, conf*100))
		} else {
			logs = append(logs, fmt.Sprintf("  • %s: %.2f (confidence %.0f%%)", in.Label, pts, conf*100))
		}
	}
	if missingCount > 0 && cfg.OnMissing == AggregatorOnMissingTreatAsZero {
		logs = append(logs, fmt.Sprintf("  (%d missing input(s) treated as zero)", missingCount))
	}
	if missingCount > 0 && cfg.OnMissing == AggregatorOnMissingSkipAndRenormalize {
		logs = append(logs, fmt.Sprintf("  (%d missing input(s) skipped; weights renormalized)", missingCount))
	}

	out := GradeOutput{RubricScores: make(map[string]float64)}
	switch cfg.Mode {
	case AggregatorModeSum:
		out = combineSum(present)
	case AggregatorModeWeightedSum:
		out, err = combineWeightedSum(present)
		if err != nil {
			return GradeOutput{}, logs, err
		}
	case AggregatorModeAverage:
		out = combineAverage(present)
	case AggregatorModeMin:
		out = combineMin(present)
	case AggregatorModeMax:
		out = combineMax(present)
	case AggregatorModeRubricMerge:
		out, err = combineRubricMerge(present, rubric)
		if err != nil {
			return GradeOutput{}, logs, err
		}
	default:
		return GradeOutput{}, logs, fmt.Errorf("unknown aggregator mode %q", cfg.Mode)
	}

	out.Confidence = combineConfidence(present, cfg.Confidence)
	if cfg.MergeComments {
		out.Comment = mergeAggregatorComments(present, cfg.CommentSep)
	}

	out.TotalPoints = clampPoints(out.TotalPoints, maxPoints)
	if rubric != nil && len(out.RubricScores) > 0 {
		if total, rubErr := validateMergedRubricTotal(rubric, out.RubricScores); rubErr == nil {
			out.TotalPoints = clampPoints(total, maxPoints)
		}
	}

	logs = append(logs, fmt.Sprintf("→ Combined total: %.2f (confidence %.0f%%)", out.TotalPoints, out.Confidence*100))
	return out, logs, nil
}

func resolveAggregatorInputs(inputs []AggregatorInput, policy AggregatorOnMissing) ([]AggregatorInput, int, error) {
	missing := 0
	for _, in := range inputs {
		if in.Missing {
			missing++
		}
	}
	if missing > 0 && policy == AggregatorOnMissingFailItem {
		return nil, missing, fmt.Errorf("aggregator input missing and onMissing policy is failItem")
	}

	present := make([]AggregatorInput, 0, len(inputs))
	for _, in := range inputs {
		if in.Missing {
			if policy == AggregatorOnMissingTreatAsZero {
				zero := GradeOutput{TotalPoints: 0, RubricScores: map[string]float64{}, Confidence: 0}
				copyIn := in
				copyIn.Missing = false
				copyIn.Grade = &zero
				present = append(present, copyIn)
			}
			continue
		}
		present = append(present, in)
	}

	if policy == AggregatorOnMissingSkipAndRenormalize && missing > 0 && len(present) > 0 {
		weightSum := 0.0
		for _, in := range present {
			weightSum += effectiveWeight(in)
		}
		if weightSum > 0 {
			for i := range present {
				w := effectiveWeight(present[i])
				present[i].Weight = w / weightSum
			}
		}
	}
	return present, missing, nil
}

func effectiveWeight(in AggregatorInput) float64 {
	if in.Weight > 0 {
		return in.Weight
	}
	return 1
}

func combineSum(inputs []AggregatorInput) GradeOutput {
	out := GradeOutput{RubricScores: make(map[string]float64)}
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		out.TotalPoints += in.Grade.TotalPoints
		mergeRubricScores(out.RubricScores, in.Grade.RubricScores)
	}
	return out
}

func combineWeightedSum(inputs []AggregatorInput) (GradeOutput, error) {
	out := GradeOutput{RubricScores: make(map[string]float64)}
	weightSum := 0.0
	for _, in := range inputs {
		weightSum += effectiveWeight(in)
	}
	if weightSum <= 0 {
		return out, fmt.Errorf("weightedSum requires positive total weight")
	}
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		w := effectiveWeight(in) / weightSum
		out.TotalPoints += in.Grade.TotalPoints * w
		for k, v := range in.Grade.RubricScores {
			out.RubricScores[k] += v * w
		}
	}
	return out, nil
}

func combineAverage(inputs []AggregatorInput) GradeOutput {
	out := GradeOutput{RubricScores: make(map[string]float64)}
	if len(inputs) == 0 {
		return out
	}
	n := float64(len(inputs))
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		out.TotalPoints += in.Grade.TotalPoints / n
		for k, v := range in.Grade.RubricScores {
			out.RubricScores[k] += v / n
		}
	}
	return out
}

func combineMin(inputs []AggregatorInput) GradeOutput {
	out := GradeOutput{RubricScores: make(map[string]float64)}
	first := true
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		if first || in.Grade.TotalPoints < out.TotalPoints {
			out.TotalPoints = in.Grade.TotalPoints
		}
		first = false
	}
	return out
}

func combineMax(inputs []AggregatorInput) GradeOutput {
	out := GradeOutput{RubricScores: make(map[string]float64)}
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		if in.Grade.TotalPoints > out.TotalPoints {
			out.TotalPoints = in.Grade.TotalPoints
		}
	}
	return out
}

func combineRubricMerge(inputs []AggregatorInput, rubric *assignmentrubric.RubricDefinition) (GradeOutput, error) {
	out := GradeOutput{RubricScores: make(map[string]float64)}
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		for k, v := range in.Grade.RubricScores {
			if _, exists := out.RubricScores[k]; exists {
				return GradeOutput{}, fmt.Errorf("rubricMerge: duplicate criterion %s from %s", k, in.Label)
			}
			out.RubricScores[k] = v
		}
	}
	if rubric != nil && len(out.RubricScores) > 0 {
		total, err := validateMergedRubricTotal(rubric, out.RubricScores)
		if err != nil {
			return GradeOutput{}, err
		}
		out.TotalPoints = total
	} else {
		for _, v := range out.RubricScores {
			out.TotalPoints += v
		}
	}
	return out, nil
}

func validateMergedRubricTotal(rubric *assignmentrubric.RubricDefinition, scores map[string]float64) (float64, error) {
	parsed := make(map[uuid.UUID]float64, len(scores))
	for k, v := range scores {
		id, err := uuid.Parse(strings.TrimSpace(k))
		if err != nil {
			continue
		}
		parsed[id] = v
	}
	return assignmentrubric.ValidateRubricScoresForGrade(rubric, parsed)
}

func mergeRubricScores(dst map[string]float64, src map[string]float64) {
	for k, v := range src {
		dst[k] = v
	}
}

func combineConfidence(inputs []AggregatorInput, mode AggregatorConfidenceMode) float64 {
	vals := make([]float64, 0, len(inputs))
	weights := make([]float64, 0, len(inputs))
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		vals = append(vals, clampConfidence(in.Grade.Confidence))
		weights = append(weights, effectiveWeight(in))
	}
	if len(vals) == 0 {
		return 1
	}
	switch mode {
	case AggregatorConfidenceMean:
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return clampConfidence(sum / float64(len(vals)))
	case AggregatorConfidenceWeighted:
		wSum := 0.0
		cSum := 0.0
		for i, v := range vals {
			w := weights[i]
			wSum += w
			cSum += v * w
		}
		if wSum <= 0 {
			return clampConfidence(vals[0])
		}
		return clampConfidence(cSum / wSum)
	default:
		min := vals[0]
		for _, v := range vals[1:] {
			if v < min {
				min = v
			}
		}
		return clampConfidence(min)
	}
}

func mergeAggregatorComments(inputs []AggregatorInput, sep string) string {
	if sep == "" {
		sep = "\n\n"
	}
	parts := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if in.Grade == nil {
			continue
		}
		c := strings.TrimSpace(in.Grade.Comment)
		if c == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s] %s", in.Label, c))
	}
	return strings.Join(parts, sep)
}

func clampPoints(v, maxPoints float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if maxPoints > 0 && v > maxPoints {
		return maxPoints
	}
	return v
}

func clampConfidence(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// DetectRubricMergeCriterionConflicts returns duplicate criterion IDs across wired criterion graders.
func DetectRubricMergeCriterionConflicts(sourceCriterionIDs []string) []string {
	seen := make(map[string]struct{})
	dupes := make([]string, 0)
	for _, id := range sourceCriterionIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			dupes = append(dupes, id)
			continue
		}
		seen[id] = struct{}{}
	}
	return dupes
}