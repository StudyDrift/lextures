package gradingagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuilderQuizSlot describes one quiz delivery slot for the AI builder prompt so
// the model knows which `question-N` / `grade-N` handles exist.
type BuilderQuizSlot struct {
	Index        int
	Label        string
	QuestionType string
	MaxPoints    float64
}

// BuilderPromptOptions configures the workflow-builder system prompt.
type BuilderPromptOptions struct {
	IsQuiz    bool
	QuizSlots []BuilderQuizSlot
	MaxPoints float64
}

// BuilderResult is what ParseBuilderResponse returns: the parsed graph plus a
// short natural-language recap the model wrote describing what it built.
type BuilderResult struct {
	Graph   *WorkflowGraph
	Summary string
}

// builderResponse is the strict JSON envelope the model must emit.
type builderResponse struct {
	Graph   json.RawMessage `json:"graph"`
	Summary string          `json:"summary"`
}

// BuildWorkflowBuilderSystemPrompt returns the authoritative, non-editable system
// prompt that constrains the model to emit a valid grading-agent workflow graph.
func BuildWorkflowBuilderSystemPrompt(opts BuilderPromptOptions) string {
	var b strings.Builder
	b.WriteString(`You are a grading-agent workflow designer. You convert an instructor's plain-English grading rule into a node/edge graph for an automated grading canvas.

The instructor message is authoritative. If a "current graph" is provided as a second message, MODIFY it to satisfy the instruction (keep unrelated nodes/edges); otherwise build a new graph from scratch.

Respond with ONLY valid JSON (no markdown fences, no prose) using this envelope:
{
  "graph": { "version": 1, "nodes": [ ... ], "edges": [ ... ] },
  "summary": "<one short sentence describing what the graph does>"
}

GRAPH RULES:
- "version" MUST be 1.
- There MUST be exactly one node of type "output" with id "output".
- Every node has: "id" (unique string), "type", "position" {"x":<number>,"y":<number>}, "data" {object}.
- Every edge has: "id" (unique string), "source" (node id), "sourceHandle", "target" (node id), "targetHandle".
- The graph MUST be acyclic. Max 50 nodes, 100 edges.
- Lay nodes out left-to-right: inputs near x=-640, transforms in the middle, the output node near x=0. Stagger y by ~120 so nodes do not overlap.

NODE TYPES AND THEIR HANDLES (source handles are outputs, target handles are inputs):
- "studentSubmission" (data {}): output handle "submission" (the student's submission text/files).
- "quizResponses" (data {}): one output handle "question-N" per quiz slot (N is the zero-based slot index) carrying that question's student response.
- "activity" (data {"assignmentItemId": "<id>"}): output handles "content" and "rubric".
- "reference" (data {"mode":"modelAnswer|answerKey|sourceText","text":"..."}): output handle "reference".
- "rubric" (data {"source":"assignment"}): output handle "rubric".
- "ai" (data {"prompt":"<grading instructions>"}): input handle "input"; output handle "output" (an AI-produced grade). Wire a submission/quiz-question/activity/reference/rubric output into "input".
- "criterionGrader" / "grader" (data {"prompt":"..."}): input handles "submission","content","rubric"; output handles "grade","comments".
- "codeTestRunner": input handle "submission"; output handles "grade","report".
- "conditionalRouter" (data {"condition":{"field":..,"operator":..,"value":..}}): input handle "input"; output handles "then" (condition true) and "else" (condition false).
- "setScore" (data {"score":<number>,"comment":"<optional>"}): input handle "grade"; output handle "grade". Use this to assign a fixed number of points on a branch.
- "scoreAggregator" (data {"mode":"sum|weightedSum|average|min|max"}): input handle "grade" (multiple); output handles "grade","comments".
- "originality","flagForReview","humanReviewGate": advanced; only use if explicitly requested.
- "output" (data {}): input handle "grade" (the final grade for an assignment), input handle "comments", and for quizzes input handles "grade-N" (the final grade for quiz slot N). Each "comments" / "grade-N" slot accepts at most one inbound edge.

CONDITIONAL ROUTER:
- "condition.field" is one of: "submissionLength","wordCount","isEmpty","score","confidence","originalityScore","isLate","submissionText","matchesRegex".
- "condition.operator" is one of: "<","<=","==",">=",">","isTrue","contains","matchesRegex".
- "isEmpty" and "isLate" only support operator "isTrue" with value true.
- "score" and "confidence" compare the numeric grade flowing into the router (value is a number).
- "submissionText"/"matchesRegex" operate on the input text: use operator "contains" or "matchesRegex" with a string value.
- Router input "input" may come from a "studentSubmission".submission, a "quizResponses".question-N, an "ai".output, a grader/codeTestRunner grade, or another router's "then"/"else".
- For tiered rules (e.g. full / partial / half / none), chain routers: the "else" branch of one router feeds the "input" of the next router, and each matching branch feeds a "setScore" whose "grade" output goes to the output node.

SET SCORE -> OUTPUT:
- A "setScore".grade output connects to the output node's "grade" slot (assignment) or "grade-N" slot (quiz slot N).
- A "conditionalRouter" "then"/"else" branch may also connect directly to the output "grade"/"grade-N" slot, or into a "setScore" input first.

`)
	if opts.IsQuiz {
		b.WriteString("ITEM CONTEXT: This is a QUIZ. Include exactly one \"quizResponses\" node with id \"quizResponses\". Available quiz slots:\n")
		if len(opts.QuizSlots) == 0 {
			b.WriteString("  (none registered yet)\n")
		}
		for _, s := range opts.QuizSlots {
			label := strings.TrimSpace(s.Label)
			if label == "" {
				label = fmt.Sprintf("Question %d", s.Index+1)
			}
			fmt.Fprintf(&b, "  - slot index %d: %s (type %s, max points %.2f) -> source handle \"question-%d\", final grade slot \"grade-%d\"\n",
				s.Index, label, strings.TrimSpace(s.QuestionType), s.MaxPoints, s.Index, s.Index)
		}
	} else {
		b.WriteString("ITEM CONTEXT: This is an ASSIGNMENT. The final grade flows into the output node's \"grade\" slot.")
		if opts.MaxPoints > 0 {
			fmt.Fprintf(&b, " The assignment is worth %.2f points.", opts.MaxPoints)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ParseBuilderResponse parses the model's JSON envelope into a normalized graph
// and the summary. It tolerates a stray ```json fence even though the model is
// told not to emit one.
func ParseBuilderResponse(raw string) (BuilderResult, error) {
	cleaned := stripJSONFence(raw)
	if cleaned == "" {
		return BuilderResult{}, fmt.Errorf("empty model response")
	}
	var env builderResponse
	if err := json.Unmarshal([]byte(cleaned), &env); err != nil {
		return BuilderResult{}, fmt.Errorf("model did not return valid JSON: %w", err)
	}
	if len(env.Graph) == 0 {
		return BuilderResult{}, fmt.Errorf("model response is missing a graph")
	}
	graph, err := UnmarshalWorkflowGraph(env.Graph)
	if err != nil {
		return BuilderResult{}, err
	}
	if graph == nil {
		return BuilderResult{}, fmt.Errorf("model returned an empty graph")
	}
	return BuilderResult{Graph: graph, Summary: strings.TrimSpace(env.Summary)}, nil
}

// stripJSONFence removes a leading/trailing markdown code fence if present and
// trims to the outermost JSON object.
func stripJSONFence(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimPrefix(s, "json")
		s = strings.TrimPrefix(s, "JSON")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
