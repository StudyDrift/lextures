package gradingagent

import (
	"context"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

type stubCodeRunner struct {
	resp codeexecution.RunResponse
	err  error
}

func (s stubCodeRunner) RunTests(_ context.Context, _ codeexecution.RunRequest) (codeexecution.RunResponse, error) {
	return s.resp, s.err
}

func sampleGraphWithCodeTestRunner() WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{
				ID: "sub1", Type: NodeTypeStudentSubmission, Position: map[string]any{"x": -640, "y": 0}, Data: map[string]any{},
			},
			{
				ID: "ctr1", Type: NodeTypeCodeTestRunner, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{
					"runtime": "python3.12",
					"mapping": map[string]any{"type": "linear", "maxPoints": 10.0},
					"testCases": []any{
						map[string]any{"id": "t1", "input": "", "expectedOutput": "4"},
						map[string]any{"id": "t2", "input": "", "expectedOutput": "4"},
					},
				},
			},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub1", SourceHandle: HandleSubmission, Target: "ctr1", TargetHandle: HandleSubmission},
			{ID: "e2", Source: "ctr1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleGrade},
			{ID: "e3", Source: "ctr1", SourceHandle: HandleReport, Target: "output", TargetHandle: HandleComments},
		},
	}
}

func TestExecuteWorkflowDryRun_codeTestRunner(t *testing.T) {
	g := sampleGraphWithCodeTestRunner()
	preview, err := ExecuteWorkflowDryRun(context.Background(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{"print(4)"},
		MaxPoints:   10,
		CodeRunner: stubCodeRunner{resp: codeexecution.RunResponse{Results: []codeexecution.TestResult{
			{TestCaseID: "t1", Passed: true, Status: codeexecution.StatusPass, ExpectedOutput: "4", ActualOutput: "4"},
			{TestCaseID: "t2", Passed: false, Status: codeexecution.StatusFail, ExpectedOutput: "4", ActualOutput: "2"},
		}}},
		Emit: func(DryRunEvent) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.SuggestedPoints != 5 {
		t.Fatalf("preview points = %v want 5", preview.SuggestedPoints)
	}
	if preview.Confidence != 1 {
		t.Fatalf("confidence = %v", preview.Confidence)
	}
	if !strings.Contains(preview.Comment, "1/2 tests passed") {
		t.Fatalf("comment = %q", preview.Comment)
	}
}

func TestValidateWorkflowGraph_codeTestRunnerRequiresTests(t *testing.T) {
	g := sampleGraphWithCodeTestRunner()
	g.Nodes[2].Data = map[string]any{"runtime": "python3.12"}
	if err := ValidateWorkflowGraph(&g); err == nil {
		t.Fatal("expected validation error for missing test cases")
	}
}
