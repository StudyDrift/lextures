package gradingagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

// CodeTestRunner executes submission code against a test suite in the sandbox.
type CodeTestRunner interface {
	RunTests(ctx context.Context, req codeexecution.RunRequest) (codeexecution.RunResponse, error)
}

type codeTestRunnerConfig struct {
	Runtime        string
	TestSuiteID    string
	TestCases      []codeexecution.TestCase
	Mapping        PassRateMapping
	Policy         PassRatePolicy
}

func parseCodeTestRunnerNodeData(data map[string]any) (codeTestRunnerConfig, error) {
	cfg := codeTestRunnerConfig{
		Runtime: "python3.12",
		Mapping: PassRateMapping{Type: "linear", MaxPoints: 10},
		Policy: PassRatePolicy{
			OnCompileError: "zero",
			OnTimeout:      "zero",
		},
	}
	if data == nil {
		return cfg, ValidationError{Field: "testCases", Message: "Add at least one test case or select a test suite."}
	}
	if v, ok := data["runtime"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.Runtime = strings.TrimSpace(v)
	}
	if v, ok := data["testSuiteId"].(string); ok {
		cfg.TestSuiteID = strings.TrimSpace(v)
	}
	if v, ok := data["onCompileError"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.Policy.OnCompileError = strings.TrimSpace(v)
	}
	if v, ok := data["onTimeout"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.Policy.OnTimeout = strings.TrimSpace(v)
	}
	if raw, ok := data["mapping"].(map[string]any); ok {
		if t, ok := raw["type"].(string); ok && strings.TrimSpace(t) != "" {
			cfg.Mapping.Type = strings.TrimSpace(t)
		}
		if mp, ok := raw["maxPoints"].(float64); ok && mp > 0 {
			cfg.Mapping.MaxPoints = mp
		}
		if weights, ok := raw["weights"].(map[string]any); ok {
			cfg.Mapping.Weights = make(map[string]float64, len(weights))
			for k, v := range weights {
				if f, ok := v.(float64); ok {
					cfg.Mapping.Weights[k] = f
				}
			}
		}
	}
	cfg.TestCases = parseInlineTestCases(data["testCases"])
	if len(cfg.TestCases) == 0 && cfg.TestSuiteID == "" {
		return cfg, ValidationError{Field: "testCases", Message: "Add at least one test case or select a test suite."}
	}
	return cfg, nil
}

func parseInlineTestCases(raw any) []codeexecution.TestCase {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]codeexecution.TestCase, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tc := codeexecution.TestCase{ID: fmt.Sprintf("t%d", i+1)}
		if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
			tc.ID = strings.TrimSpace(id)
		}
		if v, ok := m["input"].(string); ok {
			tc.Input = v
		}
		if v, ok := m["expectedOutput"].(string); ok {
			tc.ExpectedOutput = v
		}
		if v, ok := m["isHidden"].(bool); ok {
			tc.IsHidden = v
		}
		if v, ok := m["timeLimitMs"].(float64); ok && v > 0 {
			tc.TimeLimitMs = int(v)
		}
		if v, ok := m["memoryLimitKb"].(float64); ok && v > 0 {
			tc.MemoryLimitKb = int(v)
		}
		out = append(out, tc)
	}
	return out
}

func codeTestRunnerHasConfig(node WorkflowNode) bool {
	cfg, err := parseCodeTestRunnerNodeData(node.Data)
	return err == nil && (len(cfg.TestCases) > 0 || cfg.TestSuiteID != "")
}

func executeCodeTestRunnerNode(
	ctx context.Context,
	node WorkflowNode,
	g *WorkflowGraph,
	nodeByID map[string]WorkflowNode,
	state *executionState,
	runner CodeTestRunner,
	emit func(DryRunEvent),
	label string,
) error {
	if runner == nil {
		return fmt.Errorf("code execution service is not configured")
	}
	cfg, err := parseCodeTestRunnerNodeData(node.Data)
	if err != nil {
		return err
	}
	submissionText := gatherSubmissionInput(g, node.ID, nodeByID, state)
	if strings.TrimSpace(submissionText) == "" {
		return ValidationError{Field: "node:" + node.ID, Message: "Connect a submission input before running code tests."}
	}
	tests := cfg.TestCases
	if len(tests) == 0 {
		return ValidationError{Field: "node:" + node.ID + ".testSuiteId", Message: "Test suite not found. Add inline test cases until suite lookup is available."}
	}

	emit(DryRunEvent{Type: "log", Level: "info", Message: fmt.Sprintf("[%s] Running %d test(s)…", label, len(tests))})
	resp, runErr := runner.RunTests(ctx, codeexecution.RunRequest{
		Runtime: cfg.Runtime,
		Code:    submissionText,
		Tests:   tests,
	})
	if runErr != nil {
		return runErr
	}
	for i, r := range resp.Results {
		statusLabel := r.Status
		if r.Passed {
			statusLabel = "pass"
		}
		emit(DryRunEvent{
			Type: "log", Level: "info",
			Message: fmt.Sprintf("[%s] Test %d/%d: %s", label, i+1, len(resp.Results), statusLabel),
		})
	}

	grade, passRate, mapErr := MapPassRateToGrade(resp.Results, cfg.Mapping, cfg.Policy)
	if mapErr != nil {
		return mapErr
	}
	report := FormatTestReport(resp.Results)
	if strings.TrimSpace(grade.Comment) == "" {
		grade.Comment = report
	}
	state.set(node.ID, HandleGrade, slotValue{grade: &grade, text: fmt.Sprintf("%.2f", grade.TotalPoints)})
	state.set(node.ID, HandleReport, slotValue{text: report})
	state.set(node.ID, HandleScore, slotValue{text: fmt.Sprintf("%.4f", passRate)})

	emit(DryRunEvent{
		Type: "log", Level: "info",
		Message: fmt.Sprintf("[%s] Test score %.2f (pass rate %.0f%%).", label, grade.TotalPoints, passRate*100),
	})
	return nil
}

func gatherSubmissionInput(g *WorkflowGraph, nodeID string, nodeByID map[string]WorkflowNode, state *executionState) string {
	for _, e := range g.Edges {
		if e.Target != nodeID || strings.TrimSpace(e.TargetHandle) != HandleSubmission {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		if v, ok := state.get(src.ID, HandleSubmission); ok && strings.TrimSpace(v.text) != "" {
			return v.text
		}
	}
	return ""
}

// WorkflowUsesLLM reports whether the graph contains LLM-backed nodes.
func WorkflowUsesLLM(g *WorkflowGraph) bool {
	return workflowUsesLLM(g)
}

func workflowUsesLLM(g *WorkflowGraph) bool {
	if g == nil {
		return false
	}
	for _, n := range g.Nodes {
		switch n.Type {
		case NodeTypeGrader, NodeTypeAI:
			return true
		}
	}
	return false
}
