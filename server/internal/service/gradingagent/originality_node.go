package gradingagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OriginalityMetric string

const (
	OriginalityMetricSimilarity   OriginalityMetric = "similarity"
	OriginalityMetricAILikelihood OriginalityMetric = "aiLikelihood"
)

// OriginalityReportRow is a stored originality report snapshot for workflow execution.
type OriginalityReportRow struct {
	Provider      string
	Status        string
	SimilarityPct *float64
	AIProbability *float64
	ReportURL     *string
	UpdatedAt     time.Time
}

// OriginalitySignal is the resolved integrity signal for one submission + metric.
type OriginalitySignal struct {
	Present   bool
	Score     *float64
	Report    string
	Flag      bool
	UpdatedAt *time.Time
	Metric    OriginalityMetric
}

func originalityMetricFromNode(n WorkflowNode) OriginalityMetric {
	if n.Data == nil {
		return OriginalityMetricSimilarity
	}
	if v, ok := n.Data["metric"].(string); ok {
		switch OriginalityMetric(strings.TrimSpace(v)) {
		case OriginalityMetricSimilarity, OriginalityMetricAILikelihood:
			return OriginalityMetric(strings.TrimSpace(v))
		}
	}
	return OriginalityMetricSimilarity
}

func originalityFlagThresholdFromNode(n WorkflowNode) float64 {
	if n.Data == nil {
		return 0.4
	}
	switch v := n.Data["flagThreshold"].(type) {
	case float64:
		if v >= 0 && v <= 1 {
			return v
		}
	case int:
		if v >= 0 && v <= 1 {
			return float64(v)
		}
	}
	return 0.4
}

func normalizeOriginalityScore(raw float64) float64 {
	if raw > 1 {
		return raw / 100
	}
	if raw < 0 {
		return 0
	}
	if raw > 1 {
		return 1
	}
	return raw
}

func bestDoneOriginalityReport(rows []OriginalityReportRow) *OriginalityReportRow {
	var best *OriginalityReportRow
	for i := range rows {
		r := &rows[i]
		if r.Status != "done" {
			continue
		}
		if best == nil || r.UpdatedAt.After(best.UpdatedAt) {
			best = r
		}
	}
	return best
}

func metricValueFromReport(metric OriginalityMetric, report *OriginalityReportRow) *float64 {
	if report == nil {
		return nil
	}
	switch metric {
	case OriginalityMetricSimilarity:
		return report.SimilarityPct
	case OriginalityMetricAILikelihood:
		return report.AIProbability
	default:
		return nil
	}
}

func formatOriginalityReport(metric OriginalityMetric, report *OriginalityReportRow, signal OriginalitySignal) string {
	if !signal.Present || report == nil {
		return "No originality report available for this submission."
	}
	metricLabel := "Similarity"
	if metric == OriginalityMetricAILikelihood {
		metricLabel = "AI-likelihood"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s signal", metricLabel)
	if signal.Score != nil {
		fmt.Fprintf(&b, ": %.0f%%", *signal.Score*100)
	}
	if !report.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, " (report %s)", report.UpdatedAt.UTC().Format("2006-01-02"))
	}
	if report.Provider != "" {
		fmt.Fprintf(&b, " · provider %s", report.Provider)
	}
	if report.ReportURL != nil && strings.TrimSpace(*report.ReportURL) != "" {
		fmt.Fprintf(&b, "\nReport: %s", strings.TrimSpace(*report.ReportURL))
	}
	return strings.TrimSpace(b.String())
}

// ResolveOriginalitySignal reads stored reports and resolves score/report/flag for a node.
func ResolveOriginalitySignal(metric OriginalityMetric, threshold float64, rows []OriginalityReportRow) OriginalitySignal {
	best := bestDoneOriginalityReport(rows)
	raw := metricValueFromReport(metric, best)
	if raw == nil {
		return OriginalitySignal{
			Present: false,
			Report:  "No originality report available for this submission.",
			Flag:    false,
			Metric:  metric,
		}
	}
	normalized := normalizeOriginalityScore(*raw)
	updated := best.UpdatedAt
	return OriginalitySignal{
		Present:   true,
		Score:     &normalized,
		Report:    formatOriginalityReport(metric, best, OriginalitySignal{Present: true, Score: &normalized, Metric: metric}),
		Flag:      normalized >= threshold,
		UpdatedAt: &updated,
		Metric:    metric,
	}
}

func isOriginalityNodeType(nodeType string) bool {
	return nodeType == NodeTypeOriginality
}

func originalityInputSourceIsValid(src WorkflowNode, srcHandle, tgtHandle string) bool {
	return tgtHandle == HandleSubmission && quizSubmissionSourceValid(src, srcHandle)
}

func originalityHasSubmissionInput(g *WorkflowGraph, nodeID string) bool {
	for _, e := range g.Edges {
		if e.Target == nodeID && strings.TrimSpace(e.TargetHandle) == HandleSubmission {
			return true
		}
	}
	return false
}

func loadOriginalityRows(ctx context.Context, in ExecutionInput) ([]OriginalityReportRow, error) {
	if in.LoadOriginalityReports == nil || in.SubmissionID == uuid.Nil {
		return nil, nil
	}
	return in.LoadOriginalityReports(ctx, in.SubmissionID)
}

func executeOriginalityNode(
	ctx context.Context,
	node WorkflowNode,
	in ExecutionInput,
	state *executionState,
	emit func(ExecutionEvent),
	label string,
) error {
	rows, err := loadOriginalityRows(ctx, in)
	if err != nil {
		return err
	}
	metric := originalityMetricFromNode(node)
	threshold := originalityFlagThresholdFromNode(node)
	signal := ResolveOriginalitySignal(metric, threshold, rows)
	best := bestDoneOriginalityReport(rows)

	if signal.Present && signal.Score != nil {
		score := *signal.Score
		state.set(node.ID, HandleScore, slotValue{score: &score, text: fmt.Sprintf("%.4f", score)})
		flag := signal.Flag
		state.set(node.ID, HandleFlag, slotValue{flag: &flag, text: fmt.Sprintf("%v", flag)})
	} else {
		state.set(node.ID, HandleScore, slotValue{text: "no report available"})
		flag := false
		state.set(node.ID, HandleFlag, slotValue{flag: &flag, text: "false"})
	}
	state.set(node.ID, HandleReport, slotValue{text: signal.Report})

	if signal.Present && signal.Score != nil {
		emit(ExecutionEvent{
			Type: "log", Level: "info",
			Message: fmt.Sprintf("[%s] %s %.2f (flag=%v)", label, signal.Metric, *signal.Score, signal.Flag),
		})
	} else {
		emit(ExecutionEvent{
			Type: "log", Level: "info",
			Message: fmt.Sprintf("[%s] No stored originality report available.", label),
		})
	}
	_ = best
	return nil
}