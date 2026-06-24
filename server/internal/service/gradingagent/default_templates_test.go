package gradingagent

import (
	"strings"
	"testing"
)

func TestDefaultTemplates_validate(t *testing.T) {
	for _, spec := range DefaultTemplates() {
		if err := ValidateWorkflowGraph(&spec.Graph); err != nil {
			t.Fatalf("template %q: %v", spec.Name, err)
		}
	}
}

func TestParticipationWorkflowGraph_emptySubmissionScoresZero(t *testing.T) {
	g := ParticipationWorkflowGraph()
	preview, err := ExecuteWorkflowDryRun(t.Context(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{""},
		MaxPoints:   10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.SuggestedPoints != 0 {
		t.Fatalf("expected 0 points for empty submission, got %v", preview.SuggestedPoints)
	}
}

func TestParticipationWorkflowGraph_nonEmptySubmissionScoresMax(t *testing.T) {
	g := ParticipationWorkflowGraph()
	preview, err := ExecuteWorkflowDryRun(t.Context(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{"I completed the reading."},
		MaxPoints:   10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.SuggestedPoints != 10 {
		t.Fatalf("expected full credit for non-empty submission, got %v", preview.SuggestedPoints)
	}
}

func TestAIGraderWorkflowGraph_usesLLM(t *testing.T) {
	g := AIGraderWorkflowGraph()
	if !WorkflowUsesLLM(&g) {
		t.Fatal("expected AI grader template to use LLM")
	}
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatal(err)
	}
}

func TestAIGraderWorkflowGraph_promptUsesWiredVariables(t *testing.T) {
	g := AIGraderWorkflowGraph()
	var prompt string
	for _, n := range g.Nodes {
		if n.ID == "ai" {
			prompt, _ = n.Data["prompt"].(string)
			break
		}
	}
	for _, want := range []string{
		"$StudentSubmission.Submissions",
		"$Activity.Content",
		"$Activity.Rubric",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("AI grader prompt missing %q: %q", want, prompt)
		}
	}
}

func TestCompileWorkflowGraph_participationTemplate(t *testing.T) {
	g := ParticipationWorkflowGraph()
	compiled, err := CompileWorkflowGraph(&g, "I submitted my work.")
	if err != nil {
		t.Fatal(err)
	}
	if compiled.GradeSource != "router" {
		t.Fatalf("expected router grade source, got %q", compiled.GradeSource)
	}
}
