package gradingagent

import (
	"context"
	"strings"
	"testing"
)

type stubDryRunRunner struct {
	scoreOut ScoreResult
}

func (s stubDryRunRunner) Score(_ context.Context, _ ScoreRequest) (ScoreResult, error) {
	return s.scoreOut, nil
}

func (s stubDryRunRunner) RunPrompt(_ context.Context, _, _, _, _ string, _ bool) (string, int, int, error) {
	return `{"total":8,"comment":"AI says 8/10","confidence":0.8,"rubric":{}}`, 10, 5, nil
}

func TestTopologicalNodeOrder_respectsDependencies(t *testing.T) {
	g := sampleGraphWithGrader("Grade fairly", true, true)
	order, err := TopologicalNodeOrder(&g)
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]int, len(order))
	for i, id := range order {
		index[id] = i
	}
	if index["g1"] >= index["output"] {
		t.Fatalf("grader should run before output: %v", order)
	}
}

func TestExecuteWorkflow_emitsCompiledPromptForAINode(t *testing.T) {
	g := sampleGraphWithAI("Summarize $Activity.Content")
	var completeEvent ExecutionEvent
	_, err := ExecuteWorkflow(context.Background(), ExecutionInput{
		Graph:          &g,
		Submissions:     []string{"Essay body"},
		MaxPoints:       100,
		ModelID:         "test/model",
		DefaultMarkdown: "Assignment instructions",
		Runner:         stubDryRunRunner{},
		Emit: func(ev ExecutionEvent) {
			if ev.Type == "node_complete" && ev.NodeID == "ai1" {
				completeEvent = ev
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if completeEvent.CompiledPrompt != "Summarize Assignment instructions" {
		t.Fatalf("compiled prompt = %q", completeEvent.CompiledPrompt)
	}
	if !strings.Contains(completeEvent.CompiledOutput, `"total":8`) {
		t.Fatalf("compiled output = %q", completeEvent.CompiledOutput)
	}
	if completeEvent.CompiledSystemPrompt == "" {
		t.Fatal("expected compiled system prompt for AI node")
	}
}

func TestExecuteWorkflow_logsOutputWithoutPersisting(t *testing.T) {
	g := sampleGraphWithGrader("Grade fairly", false, false)
	var logs []string
	var nodeStarts []string
	preview, err := ExecuteWorkflow(context.Background(), ExecutionInput{
		Graph:          &g,
		Submissions: []string{"Essay body"},
		MaxPoints:   100,
		ModelID:     "test/model",
		Runner: stubDryRunRunner{
			scoreOut: ScoreResult{
				Output: GradeOutput{
					TotalPoints:  88,
					Comment:      "Strong thesis.",
					Confidence:   0.9,
					RubricScores: nil,
				},
				ModelID: "test/model",
			},
		},
		Emit: func(ev ExecutionEvent) {
			switch ev.Type {
			case "log":
				logs = append(logs, ev.Message)
			case "node_start":
				nodeStarts = append(nodeStarts, ev.NodeID)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.SuggestedPoints != 88 {
		t.Fatalf("preview points = %v", preview.SuggestedPoints)
	}
	if len(nodeStarts) == 0 || nodeStarts[len(nodeStarts)-1] != "output" {
		t.Fatalf("expected output node activation, got %v", nodeStarts)
	}
	foundConsole := false
	for _, line := range logs {
		if line == "── Student Grade (dry run — not persisted) ──" {
			foundConsole = true
		}
	}
	if !foundConsole {
		t.Fatalf("expected console header in logs: %v", logs)
	}
}