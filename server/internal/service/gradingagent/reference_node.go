package gradingagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const maxReferenceTextChars = 20_000

type ReferenceMode string

const (
	ReferenceModeModelAnswer ReferenceMode = "modelAnswer"
	ReferenceModeAnswerKey   ReferenceMode = "answerKey"
	ReferenceModeSourceText  ReferenceMode = "sourceText"
)

func isReferenceNodeType(nodeType string) bool {
	return nodeType == NodeTypeReference
}

func referenceModeFromNode(n WorkflowNode) ReferenceMode {
	if n.Data == nil {
		return ReferenceModeModelAnswer
	}
	switch strings.TrimSpace(fmt.Sprint(n.Data["mode"])) {
	case string(ReferenceModeAnswerKey):
		return ReferenceModeAnswerKey
	case string(ReferenceModeSourceText):
		return ReferenceModeSourceText
	default:
		return ReferenceModeModelAnswer
	}
}

func referenceInlineText(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	if v, ok := n.Data["text"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func referenceResourceID(n WorkflowNode) (uuid.UUID, bool) {
	if n.Data == nil {
		return uuid.Nil, false
	}
	raw, ok := n.Data["resourceId"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func referenceHasSource(n WorkflowNode) bool {
	if referenceInlineText(n) != "" {
		return true
	}
	_, ok := referenceResourceID(n)
	return ok
}

func referenceTrustedLabel(mode ReferenceMode) string {
	switch mode {
	case ReferenceModeAnswerKey:
		return "Answer Key (reference — trusted)"
	case ReferenceModeSourceText:
		return "Source Text (reference — trusted)"
	default:
		return "Model Answer (reference — trusted)"
	}
}

func formatReferenceTrustedAIBlock(src WorkflowNode, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return fmt.Sprintf("## %s\n%s", referenceTrustedLabel(referenceModeFromNode(src)), text)
}

func truncateReferenceText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if len(text) <= maxReferenceTextChars {
		return text, false
	}
	return text[:maxReferenceTextChars] + "\n\n[Reference truncated at 20,000 characters.]", true
}

// LoadReferenceText resolves inline text or extracts text from a course file.
func (in ExecutionInput) LoadReferenceText(ctx context.Context, node WorkflowNode) (string, bool, error) {
	if resourceID, ok := referenceResourceID(node); ok {
		if in.LoadReferenceFile == nil {
			return "", false, fmt.Errorf("reference file loader not configured")
		}
		raw, err := in.LoadReferenceFile(ctx, in.CourseCode, resourceID)
		if err != nil {
			return "", false, ValidationError{Field: "node:" + node.ID + ".resourceId", Message: "Could not extract text from the selected file."}
		}
		text, truncated := truncateReferenceText(raw)
		if text == "" {
			return "", false, ValidationError{Field: "node:" + node.ID + ".resourceId", Message: "Selected file has no extractable text."}
		}
		return text, truncated, nil
	}
	inline := referenceInlineText(node)
	if inline != "" {
		text, truncated := truncateReferenceText(inline)
		return text, truncated, nil
	}
	return "", false, ValidationError{Field: "node:" + node.ID + ".text", Message: "Add reference text or select a course file."}
}

func referenceContentSourceIsValid(src WorkflowNode, srcHandle string) bool {
	return isReferenceNodeType(src.Type) && srcHandle == HandleReference
}