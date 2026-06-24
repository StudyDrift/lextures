package gradingagent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

type RubricSourceMode string

const (
	RubricSourceAssignment RubricSourceMode = "assignment"
	RubricSourceLibrary    RubricSourceMode = "library"
	RubricSourceInline     RubricSourceMode = "inline"
)

func isRubricNodeType(nodeType string) bool {
	return nodeType == NodeTypeRubric
}

func rubricSourceFromNode(n WorkflowNode) RubricSourceMode {
	if n.Data == nil {
		return RubricSourceAssignment
	}
	switch strings.TrimSpace(fmt.Sprint(n.Data["source"])) {
	case string(RubricSourceLibrary):
		return RubricSourceLibrary
	case string(RubricSourceInline):
		return RubricSourceInline
	default:
		return RubricSourceAssignment
	}
}

func rubricLibraryAssignmentItemID(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	if v, ok := n.Data["rubricAssignmentItemId"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func rubricHasSource(n WorkflowNode) bool {
	switch rubricSourceFromNode(n) {
	case RubricSourceInline:
		rubric, _ := parseInlineRubric(n)
		return rubric != nil && len(rubric.Criteria) > 0
	case RubricSourceLibrary:
		return rubricLibraryAssignmentItemID(n) != ""
	default:
		return true
	}
}

func parseInlineRubric(n WorkflowNode) (*assignmentrubric.RubricDefinition, error) {
	if n.Data == nil {
		return nil, nil
	}
	raw, ok := n.Data["rubric"]
	if !ok || raw == nil {
		return nil, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var def assignmentrubric.RubricDefinition
	if err := json.Unmarshal(b, &def); err != nil {
		return nil, err
	}
	if len(def.Criteria) == 0 {
		return nil, nil
	}
	for i := range def.Criteria {
		if def.Criteria[i].ID == uuid.Nil {
			return nil, fmt.Errorf("inline rubric criterion missing id")
		}
	}
	return &def, nil
}

// LoadRubricDefinition resolves rubric data for a Rubric node.
func (in DryRunExecutionInput) LoadRubricDefinition(node WorkflowNode) (*assignmentrubric.RubricDefinition, error) {
	switch rubricSourceFromNode(node) {
	case RubricSourceInline:
		rubric, err := parseInlineRubric(node)
		if err != nil {
			return nil, ValidationError{Field: "node:" + node.ID + ".rubric", Message: "Inline rubric is invalid."}
		}
		if rubric == nil {
			return nil, ValidationError{Field: "node:" + node.ID + ".rubric", Message: "Add at least one rubric criterion."}
		}
		return rubric, nil
	case RubricSourceLibrary:
		itemID := rubricLibraryAssignmentItemID(node)
		if itemID == "" {
			return nil, ValidationError{Field: "node:" + node.ID + ".rubricAssignmentItemId", Message: "Select an assignment with a rubric."}
		}
		_, rubric, err := in.resolveActivity(itemID)
		if err != nil {
			return nil, err
		}
		if rubric == nil || len(rubric.Criteria) == 0 {
			return nil, ValidationError{Field: "node:" + node.ID + ".rubricAssignmentItemId", Message: "Selected assignment has no rubric."}
		}
		return rubric, nil
	default:
		_, rubric, err := in.resolveActivity("")
		if err != nil {
			return nil, err
		}
		if rubric == nil || len(rubric.Criteria) == 0 {
			return nil, ValidationError{Field: "node:" + node.ID + ".source", Message: "This assignment has no rubric."}
		}
		return rubric, nil
	}
}

func rubricOutputSourceIsValid(src WorkflowNode, srcHandle string) bool {
	return isRubricNodeType(src.Type) && srcHandle == HandleRubric
}