package gradingagent

import (
	"strings"
	"testing"
)

func TestBuildWorkflowBuilderSystemPrompt_quizSlots(t *testing.T) {
	prompt := BuildWorkflowBuilderSystemPrompt(BuilderPromptOptions{
		IsQuiz: true,
		QuizSlots: []BuilderQuizSlot{
			{Index: 0, Label: "Question 1", QuestionType: "numeric", MaxPoints: 10},
			{Index: 1, Label: "Question 2", QuestionType: "essay", MaxPoints: 5},
		},
	})
	if !strings.Contains(prompt, "quizResponses") {
		t.Fatal("quiz prompt should mention the quizResponses node")
	}
	if !strings.Contains(prompt, "question-0") || !strings.Contains(prompt, "grade-0") {
		t.Fatalf("quiz prompt should describe slot handles: %q", prompt)
	}
	if !strings.Contains(prompt, "Question 1") || !strings.Contains(prompt, "Question 2") {
		t.Fatal("quiz prompt should list slot labels")
	}
}

func TestBuildWorkflowBuilderSystemPrompt_assignment(t *testing.T) {
	prompt := BuildWorkflowBuilderSystemPrompt(BuilderPromptOptions{MaxPoints: 20})
	if !strings.Contains(prompt, "ASSIGNMENT") {
		t.Fatal("assignment prompt should declare assignment context")
	}
	if !strings.Contains(prompt, "20.00 points") {
		t.Fatalf("assignment prompt should include max points: %q", prompt)
	}
}

func TestParseBuilderResponse_plainJSON(t *testing.T) {
	raw := `{"graph":{"version":1,"nodes":[{"id":"output","type":"output","position":{"x":0,"y":0},"data":{}}],"edges":[]},"summary":"Empty graph"}`
	res, err := ParseBuilderResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Graph == nil || len(res.Graph.Nodes) != 1 {
		t.Fatalf("expected one node, got %+v", res.Graph)
	}
	if res.Summary != "Empty graph" {
		t.Fatalf("unexpected summary: %q", res.Summary)
	}
}

func TestParseBuilderResponse_fencedJSON(t *testing.T) {
	raw := "```json\n{\"graph\":{\"version\":1,\"nodes\":[{\"id\":\"output\",\"type\":\"output\",\"position\":{\"x\":0,\"y\":0},\"data\":{}}],\"edges\":[]},\"summary\":\"ok\"}\n```"
	res, err := ParseBuilderResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing fenced JSON: %v", err)
	}
	if res.Graph == nil || len(res.Graph.Nodes) != 1 {
		t.Fatalf("expected one node from fenced JSON, got %+v", res.Graph)
	}
}

func TestParseBuilderResponse_rejectsGarbage(t *testing.T) {
	if _, err := ParseBuilderResponse("not json at all"); err == nil {
		t.Fatal("expected error for non-JSON response")
	}
	if _, err := ParseBuilderResponse(""); err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestParseBuilderResponse_validatesTieredQuizGraph(t *testing.T) {
	// Mirrors the canonical example: quiz question -> router -> setScore -> output(grade-0).
	raw := `{
      "graph": {
        "version": 1,
        "nodes": [
          {"id":"quizResponses","type":"quizResponses","position":{"x":-640,"y":0},"data":{}},
          {"id":"rtr1","type":"conditionalRouter","position":{"x":-320,"y":0},"data":{"condition":{"field":"submissionText","operator":"contains","value":"10"}}},
          {"id":"ss1","type":"setScore","position":{"x":-120,"y":-80},"data":{"score":10}},
          {"id":"output","type":"output","position":{"x":120,"y":0},"data":{}}
        ],
        "edges": [
          {"id":"e1","source":"quizResponses","sourceHandle":"question-0","target":"rtr1","targetHandle":"input"},
          {"id":"e2","source":"rtr1","sourceHandle":"then","target":"ss1","targetHandle":"grade"},
          {"id":"e3","source":"ss1","sourceHandle":"grade","target":"output","targetHandle":"grade-0"}
        ]
      },
      "summary": "Scores question 1"
    }`
	res, err := ParseBuilderResponse(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := ValidateWorkflowGraphForPersistence(res.Graph); err != nil {
		t.Fatalf("expected tiered quiz graph to pass persistence validation: %v", err)
	}
}
