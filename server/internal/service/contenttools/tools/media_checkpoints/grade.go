package media_checkpoints

import "github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"

// GradeResult is the outcome of grading one checkpoint response.
type GradeResult = inline_questions.GradeResult

// ToInlineQuestion adapts a checkpoint question into the CT.11 grader shape.
func ToInlineQuestion(cp Checkpoint) inline_questions.Question {
	q := inline_questions.Question{
		ID:              cp.ID,
		Type:            cp.Question.Type,
		Prompt:          cp.Question.Prompt,
		AcceptedAnswers: append([]string{}, cp.Question.AcceptedAnswers...),
		CorrectValue:    cp.Question.CorrectValue,
		Points:          1,
	}
	if len(cp.Question.Options) > 0 {
		q.Options = make([]inline_questions.Option, len(cp.Question.Options))
		for i, o := range cp.Question.Options {
			q.Options[i] = inline_questions.Option{
				ID:       o.ID,
				Text:     o.Text,
				Correct:  o.Correct,
				Feedback: o.Feedback,
			}
		}
	}
	if cp.Question.Tolerance != nil {
		q.Tolerance = &inline_questions.Tolerance{
			Kind:  cp.Question.Tolerance.Kind,
			Value: cp.Question.Tolerance.Value,
		}
	}
	return q
}

// GradeCheckpoint grades a learner value against a checkpoint definition.
func GradeCheckpoint(cp Checkpoint, value any) GradeResult {
	return inline_questions.GradeQuestion(ToInlineQuestion(cp), value)
}
