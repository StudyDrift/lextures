package gradingagent

import (
	"fmt"
	"math"
	"strings"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

// PassRateMapping configures how test pass rate maps to points.
type PassRateMapping struct {
	Type      string             // linear | allOrNothing | weighted
	MaxPoints float64
	Weights   map[string]float64 // test id -> weight (weighted only)
}

// PassRatePolicy controls grading when compile/timeout failures occur.
type PassRatePolicy struct {
	OnCompileError string // zero | failItem
	OnTimeout      string // zero | partial | failItem
}

// MapPassRateToGrade converts test results into a deterministic grade.
func MapPassRateToGrade(results []codeexecution.TestResult, mapping PassRateMapping, policy PassRatePolicy) (GradeOutput, float64, error) {
	if len(results) == 0 {
		return GradeOutput{}, 0, fmt.Errorf("no test results")
	}
	if mapping.MaxPoints <= 0 {
		mapping.MaxPoints = 10
	}
	if policy.OnCompileError == "" {
		policy.OnCompileError = "zero"
	}
	if policy.OnTimeout == "" {
		policy.OnTimeout = "zero"
	}

	for _, r := range results {
		if r.Status == codeexecution.StatusCE {
			if policy.OnCompileError == "zero" {
				return GradeOutput{
					TotalPoints: 0,
					Comment:     formatCompileFailureComment(r),
					Confidence:  1,
				}, 0, nil
			}
		}
	}

	points, passRate := scoreFromResults(results, mapping, policy)
	return GradeOutput{
		TotalPoints: points,
		Comment:     "",
		Confidence:  1,
	}, passRate, nil
}

func scoreFromResults(results []codeexecution.TestResult, mapping PassRateMapping, policy PassRatePolicy) (points, passRate float64) {
	switch mapping.Type {
	case "allOrNothing":
		for _, r := range results {
			if !resultCountsAsPass(r, policy) {
				return 0, 0
			}
		}
		return mapping.MaxPoints, 1
	case "weighted":
		var earned, total float64
		for _, r := range results {
			w := mapping.Weights[r.TestCaseID]
			if w <= 0 {
				w = 1
			}
			total += w
			if resultCountsAsPass(r, policy) {
				earned += w
			}
		}
		if total <= 0 {
			return 0, 0
		}
		passRate = earned / total
		return roundPoints(passRate * mapping.MaxPoints), passRate
	default: // linear
		var passed, counted float64
		for _, r := range results {
			if r.Status == codeexecution.StatusCE {
				continue
			}
			counted++
			if resultCountsAsPass(r, policy) {
				passed++
			}
		}
		if counted == 0 {
			return 0, 0
		}
		passRate = passed / counted
		return roundPoints(passRate * mapping.MaxPoints), passRate
	}
}

func resultCountsAsPass(r codeexecution.TestResult, policy PassRatePolicy) bool {
	if r.Passed {
		return true
	}
	if r.Status == codeexecution.StatusTLE && policy.OnTimeout == "partial" {
		return false
	}
	return false
}

func roundPoints(v float64) float64 {
	return math.Round(v*100) / 100
}

func formatCompileFailureComment(r codeexecution.TestResult) string {
	if strings.TrimSpace(r.Stderr) == "" {
		return "Compile error."
	}
	return "Compile error:\n" + r.Stderr
}

// FormatTestReport builds a human-readable per-test report.
func FormatTestReport(results []codeexecution.TestResult) string {
	if len(results) == 0 {
		return "No test results."
	}
	var b strings.Builder
	passed := 0
	for i, r := range results {
		if r.Passed {
			passed++
		}
		label := fmt.Sprintf("Test %d", i+1)
		if r.TestCaseID != "" {
			label = r.TestCaseID
		}
		status := r.Status
		if status == "" {
			if r.Passed {
				status = codeexecution.StatusPass
			} else {
				status = codeexecution.StatusFail
			}
		}
		fmt.Fprintf(&b, "[%s] %s", label, strings.ToUpper(status))
		if r.Passed {
			b.WriteString(" ✓")
		} else {
			b.WriteString(" ✗")
		}
		b.WriteByte('\n')
		if !r.Passed && strings.TrimSpace(r.ActualOutput) != "" {
			fmt.Fprintf(&b, "  actual:   %s\n", truncateLog(r.ActualOutput, 500))
		}
		if !r.Passed && strings.TrimSpace(r.ExpectedOutput) != "" {
			fmt.Fprintf(&b, "  expected: %s\n", truncateLog(r.ExpectedOutput, 500))
		}
		if strings.TrimSpace(r.Stderr) != "" {
			fmt.Fprintf(&b, "  stderr:   %s\n", truncateLog(r.Stderr, 500))
		}
		if i < len(results)-1 {
			b.WriteByte('\n')
		}
	}
	fmt.Fprintf(&b, "\n%d/%d tests passed.", passed, len(results))
	return b.String()
}
