package gradingagent

import (
	"strconv"
	"strings"
)

const (
	HandleQuestionPrefix = "question-"
	HandleQuizGradePrefix = "grade-"
)

func isQuizQuestionHandle(handle string) bool {
	_, ok := parseQuizQuestionHandle(handle)
	return ok
}

func isQuizGradeHandle(handle string) bool {
	_, ok := parseQuizGradeHandle(handle)
	return ok
}

func parseQuizQuestionHandle(handle string) (int, bool) {
	if !strings.HasPrefix(handle, HandleQuestionPrefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(handle, HandleQuestionPrefix))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func parseQuizGradeHandle(handle string) (int, bool) {
	if !strings.HasPrefix(handle, HandleQuizGradePrefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(handle, HandleQuizGradePrefix))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func graphIsQuizMode(g *WorkflowGraph) bool {
	if g == nil {
		return false
	}
	for _, n := range g.Nodes {
		if n.Type == NodeTypeQuizResponses {
			return true
		}
	}
	return false
}

func quizSubmissionSourceValid(src WorkflowNode, srcHandle string) bool {
	if isStudentSubmissionNodeType(src.Type) {
		return true
	}
	return isQuizResponsesNodeType(src.Type) && isQuizQuestionHandle(srcHandle)
}