package gradingagent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

var promptVariablePattern = regexp.MustCompile(`\$([A-Za-z][A-Za-z0-9]*)\.([A-Za-z][A-Za-z0-9]*)`)

// PromptVariableContext supplies runtime values for wired prompt variables.
type PromptVariableContext struct {
	Submissions     []string
	ContentMarkdown string
	Rubric          *assignmentrubric.RubricDefinition
	ReferenceTexts  map[string]string
}

func workflowNodeVariableName(label string) string {
	return strings.ReplaceAll(label, " ", "")
}

func workflowNodeLabel(data map[string]any) string {
	if data == nil {
		return ""
	}
	if v, ok := data["label"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func defaultNodeLabel(nodeType string) string {
	switch nodeType {
	case NodeTypeStudentSubmission:
		return "Student Submission"
	case NodeTypeActivity:
		return "Activity"
	case NodeTypeAI:
		return "AI"
	case NodeTypeGrader:
		return "Grader (LLM)"
	case NodeTypeCriterionGrader:
		return "Criterion Grader"
	case NodeTypeCodeTestRunner:
		return "Code Test Runner"
	case NodeTypeConditionalRouter:
		return "Conditional Router"
	case NodeTypeFlagForReview:
		return "Flag for Review"
	case NodeTypeHumanReviewGate:
		return "Human Review Gate"
	case NodeTypeOriginality:
		return "Originality Check"
	case NodeTypeReference:
		return "Reference Material"
	case NodeTypeRubric:
		return "Rubric"
	case NodeTypeScoreAggregator:
		return "Score Aggregator"
	case NodeTypeOutput:
		return "Student grade"
	default:
		return nodeType
	}
}

func workflowNodeDisplayLabel(data map[string]any, nodeType string) string {
	if label := workflowNodeLabel(data); label != "" {
		return label
	}
	return defaultNodeLabel(nodeType)
}

func workflowOutputHandleToProperty(handle string) string {
	switch handle {
	case HandleSubmission:
		return "Submissions"
	case HandleContent:
		return "Content"
	case HandleRubric:
		return "Rubric"
	case HandleAIOutput:
		return "Output"
	case HandleReference:
		return "Text"
	default:
		return ""
	}
}

func wiredInputTargetHandles(promptNodeType string) []string {
	switch promptNodeType {
	case NodeTypeAI:
		return []string{HandleAIInput}
	case NodeTypeGrader, NodeTypeCriterionGrader:
		return []string{HandleSubmission, HandleContent, HandleRubric}
	default:
		return nil
	}
}

func formatRubricVariableText(rubric *assignmentrubric.RubricDefinition) string {
	if rubric == nil || len(rubric.Criteria) == 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range rubric.Criteria {
		fmt.Fprintf(&b, "- %s (%s): allowed scores", c.Title, c.ID.String())
		for i, lvl := range c.Levels {
			if i == 0 {
				b.WriteString(" ")
			} else {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%.2f", lvl.Points)
		}
		b.WriteString("\n")
	}
	raw, _ := json.Marshal(rubric)
	b.Write(raw)
	return strings.TrimSpace(b.String())
}

func propertyValue(handle string, ctx PromptVariableContext) string {
	switch handle {
	case HandleSubmission:
		return JoinSubmissions(ctx.Submissions)
	case HandleContent:
		return strings.TrimSpace(ctx.ContentMarkdown)
	case HandleRubric:
		return formatRubricVariableText(ctx.Rubric)
	default:
		return ""
	}
}

func buildPromptVariableBindings(g *WorkflowGraph, promptNodeID string, ctx PromptVariableContext) map[string]map[string]string {
	if g == nil || strings.TrimSpace(promptNodeID) == "" {
		return nil
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	promptNode, ok := nodeByID[promptNodeID]
	if !ok {
		return nil
	}
	targetHandles := wiredInputTargetHandles(promptNode.Type)
	if len(targetHandles) == 0 {
		return nil
	}
	allowedTargets := make(map[string]struct{}, len(targetHandles))
	for _, h := range targetHandles {
		allowedTargets[h] = struct{}{}
	}

	type grouped struct {
		node    WorkflowNode
		handles map[string]struct{}
	}
	groupedBySource := make(map[string]*grouped)
	for _, e := range g.Edges {
		if e.Target != promptNodeID {
			continue
		}
		if _, ok := allowedTargets[strings.TrimSpace(e.TargetHandle)]; !ok {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		handle := strings.TrimSpace(e.SourceHandle)
		property := workflowOutputHandleToProperty(handle)
		if property == "" {
			continue
		}
		entry := groupedBySource[src.ID]
		if entry == nil {
			entry = &grouped{node: src, handles: make(map[string]struct{})}
			groupedBySource[src.ID] = entry
		}
		entry.handles[handle] = struct{}{}
	}

	bindings := make(map[string]map[string]string)
	for _, entry := range groupedBySource {
		varName := workflowNodeVariableName(workflowNodeDisplayLabel(entry.node.Data, entry.node.Type))
		if varName == "" {
			continue
		}
		props := bindings[varName]
		if props == nil {
			props = make(map[string]string)
			bindings[varName] = props
		}
		for handle := range entry.handles {
			property := workflowOutputHandleToProperty(handle)
			if property == "" {
				continue
			}
			if handle == HandleReference {
				if ctx.ReferenceTexts != nil {
					props[property] = ctx.ReferenceTexts[entry.node.ID]
				}
				continue
			}
			props[property] = propertyValue(handle, ctx)
		}
	}
	return bindings
}

// SubstitutePromptVariables replaces `$NodeName.Property` tokens in a prompt.
func SubstitutePromptVariables(prompt string, bindings map[string]map[string]string) string {
	if prompt == "" || len(bindings) == 0 {
		return prompt
	}
	return promptVariablePattern.ReplaceAllStringFunc(prompt, func(match string) string {
		parts := promptVariablePattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		nodeName, propertyName := parts[1], parts[2]
		nodeValues, ok := bindings[nodeName]
		if !ok {
			return match
		}
		value, ok := nodeValues[propertyName]
		if !ok {
			return match
		}
		return value
	})
}

// SubstituteWorkflowPromptVariables resolves wired variables for one prompt node.
func SubstituteWorkflowPromptVariables(g *WorkflowGraph, promptNodeID, prompt string, ctx PromptVariableContext) string {
	bindings := buildPromptVariableBindings(g, promptNodeID, ctx)
	return SubstitutePromptVariables(prompt, bindings)
}