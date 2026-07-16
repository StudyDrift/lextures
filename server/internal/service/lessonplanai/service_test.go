package lessonplanai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

type stubChat struct {
	replies map[string]string
}

func (s stubChat) Complete(ctx context.Context, model string, messages []aiprovider.Message, opts ...aiprovider.ChatOptions) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	user := ""
	for _, m := range messages {
		if m.Role == "user" {
			user = m.Content
		}
	}
	key := "markdown"
	if len(opts) > 0 && opts[0].JSONMode {
		key = "json"
	}
	if strings.Contains(user, "exit ticket") || strings.Contains(user, "formative") {
		key = "json"
	}
	if strings.Contains(user, "rubric") {
		key = "json"
	}
	text := s.replies[key]
	return aiprovider.ChatResult{Text: text}, aiprovider.CallMeta{}, nil
}

func TestValidateInput(t *testing.T) {
	if err := ValidateInput(InputParams{}); err == nil {
		t.Fatal("expected error for empty input")
	}
	if err := ValidateInput(InputParams{
		LearningObjective: "Multiply fractions",
		GradeLevel:        "5",
		Subject:           "Math",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRedactInput(t *testing.T) {
	out := RedactInput(InputParams{
		LearningObjective: "Contact me at test@example.com",
		GradeLevel:        "4",
		Subject:           "ELA",
	})
	if strings.Contains(out.LearningObjective, "test@example.com") {
		t.Fatalf("expected redacted email, got %q", out.LearningObjective)
	}
}

func TestGenerateParallelPartialFailure(t *testing.T) {
	quizJSON := `{"questions":[{"id":"q1","prompt":"What is the main idea?","questionType":"short_answer","choices":[],"points":1,"estimatedMinutes":2,"required":true}]}`
	rubricJSON := `{"criteria":[{"id":"550e8400-e29b-41d4-a716-446655440000","title":"Main idea","levels":[{"label":"Emerging","points":1},{"label":"Proficient","points":3}]}]}`
	client := stubChat{replies: map[string]string{
		"markdown": "## Lesson plan\nWarm-up and instruction.",
		"json":     quizJSON,
	}}
	client.replies["json"] = quizJSON

	input := InputParams{
		LearningObjective:     "Identify the main idea",
		GradeLevel:            "4",
		Subject:               "ELA",
		DifferentiationLevels: []string{"on_grade"},
	}
	keys := BuildComponentKeys(input.DifferentiationLevels)
	pkg := NewPackage("job-1", keys)

	pkg = Generate(context.Background(), client, input, pkg, GenerateOptions{
		ModelID: "test-model",
		Prompts: Prompts{
			LessonPlan: "plan",
			Activity:   "activity",
			Quiz:       "quiz",
			Rubric:     "rubric",
		},
		ForceFail: map[string]error{
			ComponentQuiz: context.DeadlineExceeded,
		},
	})

	var lessonDone, quizFailed bool
	for _, c := range pkg.Components {
		if c.Key == ComponentLessonPlan && c.Status == StatusCompleted {
			lessonDone = true
		}
		if c.Key == ComponentQuiz && c.Status == StatusFailed {
			quizFailed = true
		}
	}
	if !lessonDone {
		t.Fatal("expected lesson plan to complete")
	}
	if !quizFailed {
		t.Fatal("expected quiz failure")
	}
	_ = rubricJSON
}

func TestGenerateQuizAndProvenance(t *testing.T) {
	client := stubChat{replies: map[string]string{
		"json": `{"questions":[{"prompt":"Q1","questionType":"true_false","choices":["True","False"],"correctChoiceIndex":0,"points":1,"estimatedMinutes":1,"required":true}]}`,
	}}
	input := InputParams{
		LearningObjective: "Multiply fractions",
		GradeLevel:        "5",
		Subject:           "Math",
	}
	keys := []string{ComponentQuiz}
	pkg := NewPackage("job-2", keys)
	pkg = Generate(context.Background(), client, input, pkg, GenerateOptions{
		ModelID:  "test-model",
		Prompts:  Prompts{Quiz: DefaultQuizPrompt},
		OnlyKeys: []string{ComponentQuiz},
	})
	if len(pkg.Components) != 1 || pkg.Components[0].Status != StatusCompleted {
		t.Fatalf("expected completed quiz, got %+v", pkg.Components)
	}
	if pkg.Components[0].Provenance == nil || pkg.Components[0].Provenance.GeneratedBy != GeneratedBy {
		t.Fatalf("expected provenance, got %+v", pkg.Components[0].Provenance)
	}
	var qc QuizContent
	if err := json.Unmarshal(pkg.Components[0].Content, &qc); err != nil {
		t.Fatal(err)
	}
	if len(qc.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(qc.Questions))
	}
}

func TestRegenerateSingleComponent(t *testing.T) {
	client := stubChat{replies: map[string]string{
		"markdown": "## Updated plan",
	}}
	input := InputParams{
		LearningObjective: "Test objective",
		GradeLevel:        "6",
		Subject:           "Science",
	}
	pkg := NewPackage("job-3", []string{ComponentLessonPlan, ComponentQuiz})
	pkg.Components[0].Status = StatusCompleted
	pkg.Components[1].Status = StatusFailed
	errMsg := "timeout"
	pkg.Components[1].Error = &errMsg

	pkg = Generate(context.Background(), client, input, pkg, GenerateOptions{
		ModelID:  "test-model",
		Prompts:  Prompts{LessonPlan: "plan"},
		OnlyKeys: []string{ComponentLessonPlan},
	})
	if pkg.Components[0].Status != StatusCompleted {
		t.Fatalf("expected regenerated lesson plan, got %s", pkg.Components[0].Status)
	}
}
