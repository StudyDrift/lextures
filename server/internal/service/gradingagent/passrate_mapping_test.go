package gradingagent

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

func TestMapPassRateToGrade_linear(t *testing.T) {
	results := []codeexecution.TestResult{
		{Passed: true, Status: codeexecution.StatusPass},
		{Passed: true, Status: codeexecution.StatusPass},
		{Passed: false, Status: codeexecution.StatusFail},
		{Passed: false, Status: codeexecution.StatusFail},
	}
	grade, passRate, err := MapPassRateToGrade(results, PassRateMapping{Type: "linear", MaxPoints: 10}, PassRatePolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if grade.TotalPoints != 5 {
		t.Fatalf("points = %v want 5", grade.TotalPoints)
	}
	if passRate != 0.5 {
		t.Fatalf("passRate = %v want 0.5", passRate)
	}
	if grade.Confidence != 1 {
		t.Fatalf("confidence = %v", grade.Confidence)
	}
}

func TestMapPassRateToGrade_allOrNothing(t *testing.T) {
	results := []codeexecution.TestResult{
		{Passed: true, Status: codeexecution.StatusPass},
		{Passed: false, Status: codeexecution.StatusFail},
	}
	grade, passRate, err := MapPassRateToGrade(results, PassRateMapping{Type: "allOrNothing", MaxPoints: 10}, PassRatePolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if grade.TotalPoints != 0 || passRate != 0 {
		t.Fatalf("points=%v passRate=%v", grade.TotalPoints, passRate)
	}
}

func TestMapPassRateToGrade_compileErrorZero(t *testing.T) {
	results := []codeexecution.TestResult{{
		Status: codeexecution.StatusCE,
		Stderr: "SyntaxError: invalid syntax",
	}}
	grade, passRate, err := MapPassRateToGrade(results, PassRateMapping{Type: "linear", MaxPoints: 10}, PassRatePolicy{OnCompileError: "zero"})
	if err != nil {
		t.Fatal(err)
	}
	if grade.TotalPoints != 0 || passRate != 0 {
		t.Fatalf("points=%v passRate=%v", grade.TotalPoints, passRate)
	}
	if grade.Comment == "" {
		t.Fatal("expected compile comment")
	}
}

func TestFormatTestReport_listsFailures(t *testing.T) {
	report := FormatTestReport([]codeexecution.TestResult{{
		TestCaseID: "case-a",
		Passed:     false,
		Status:     codeexecution.StatusFail,
		ActualOutput: "2",
		ExpectedOutput: "4",
	}})
	if report == "" {
		t.Fatal("empty report")
	}
	if !strings.Contains(report, "case-a") || !strings.Contains(report, "FAIL") || !strings.Contains(report, "0/1") {
		t.Fatalf("report = %q", report)
	}
}
