package lessonplanai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/service/aitutor"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const (
	GeneratedBy = "lextures-ai"

	ComponentLessonPlan       = "lesson_plan"
	ComponentQuiz             = "quiz"
	ComponentRubric           = "rubric"
	ComponentActivityPrefix   = "activity_"
)

var validDifferentiationLevels = map[string]bool{
	"below_grade": true,
	"on_grade":    true,
	"advanced":    true,
	"ell":         true,
	"iep":         true,
}

// InputParams is the instructor request for lesson generation (FR-1).
type InputParams struct {
	LearningObjective     string   `json:"learning_objective"`
	GradeLevel            string   `json:"grade_level"`
	Subject               string   `json:"subject"`
	DurationMinutes       *int     `json:"duration_minutes,omitempty"`
	StandardsCode         *string  `json:"standards_code,omitempty"`
	DifferentiationLevels []string `json:"differentiation_levels,omitempty"`
}

// Provenance tags AI-generated artifacts (FR-4).
type Provenance struct {
	GeneratedBy  string `json:"generated_by"`
	ModelID      string `json:"model_id"`
	GenerationTS string `json:"generation_ts"`
}

// ComponentStatus tracks per-component generation state.
type ComponentStatus string

const (
	StatusPending    ComponentStatus = "pending"
	StatusProcessing ComponentStatus = "processing"
	StatusCompleted  ComponentStatus = "completed"
	StatusFailed     ComponentStatus = "failed"
)

// LessonPlanContent is the lesson plan outline component.
type LessonPlanContent struct {
	Markdown string `json:"markdown"`
}

// ActivityContent is a differentiated activity variant.
type ActivityContent struct {
	Level    string `json:"level"`
	Markdown string `json:"markdown"`
}

// QuizContent is the formative assessment component.
type QuizContent struct {
	Questions []coursemodulequiz.QuizQuestion `json:"questions"`
}

// RubricContent wraps an assignment rubric for open-ended tasks.
type RubricContent struct {
	Rubric assignmentrubric.RubricDefinition `json:"rubric"`
}

// ComponentSlot is one editable generated asset (FR-3).
type ComponentSlot struct {
	Key        string          `json:"key"`
	Status     ComponentStatus `json:"status"`
	Error      *string         `json:"error,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
	Provenance *Provenance     `json:"provenance,omitempty"`
}

// PackageResult is the full lesson package returned to the client.
type PackageResult struct {
	JobID               string          `json:"job_id"`
	Status              string          `json:"status"`
	Components          []ComponentSlot `json:"components"`
	StandardsDisclaimer *string         `json:"standards_disclaimer,omitempty"`
}

// ChatClient abstracts OpenRouter for tests.
type ChatClient interface {
	ChatCompletion(model string, messages []openrouter.Message, opts ...openrouter.ChatOptions) (openrouter.ChatResult, error)
}

// Prompts holds system prompts for generation sub-tasks.
type Prompts struct {
	LessonPlan string
	Activity   string
	Quiz       string
	Rubric     string
}

// GenerateOptions configures a full or partial generation run.
type GenerateOptions struct {
	ModelID    string
	Prompts    Prompts
	OnlyKeys   []string // empty = all components
	ForceFail  map[string]error // test hook: force component failure
}

// ValidateInput checks required fields on instructor input.
func ValidateInput(p InputParams) error {
	if strings.TrimSpace(p.LearningObjective) == "" {
		return fmt.Errorf("learning_objective is required")
	}
	if strings.TrimSpace(p.GradeLevel) == "" {
		return fmt.Errorf("grade_level is required")
	}
	if strings.TrimSpace(p.Subject) == "" {
		return fmt.Errorf("subject is required")
	}
	for _, lvl := range p.DifferentiationLevels {
		if !validDifferentiationLevels[lvl] {
			return fmt.Errorf("invalid differentiation level: %s", lvl)
		}
	}
	return nil
}

// RedactInput applies PII redaction to instructor-supplied text (FR-8 / 19.11).
func RedactInput(p InputParams) InputParams {
	out := p
	out.LearningObjective = aitutor.RedactPII(strings.TrimSpace(p.LearningObjective))
	out.GradeLevel = strings.TrimSpace(p.GradeLevel)
	out.Subject = strings.TrimSpace(p.Subject)
	if p.StandardsCode != nil {
		s := aitutor.RedactPII(strings.TrimSpace(*p.StandardsCode))
		out.StandardsCode = &s
	}
	return out
}

// BuildComponentKeys returns the component keys for a generation request.
func BuildComponentKeys(levels []string) []string {
	keys := []string{ComponentLessonPlan, ComponentQuiz, ComponentRubric}
	if len(levels) == 0 {
		levels = []string{"on_grade"}
	}
	for _, lvl := range levels {
		keys = append(keys, ComponentActivityPrefix+lvl)
	}
	return keys
}

// NewPackage initializes a package with pending component slots.
func NewPackage(jobID string, keys []string) PackageResult {
	slots := make([]ComponentSlot, 0, len(keys))
	for _, k := range keys {
		slots = append(slots, ComponentSlot{Key: k, Status: StatusPending})
	}
	return PackageResult{
		JobID:      jobID,
		Status:     "processing",
		Components: slots,
	}
}

func shouldGenerate(key string, only []string) bool {
	if len(only) == 0 {
		return true
	}
	for _, k := range only {
		if k == key {
			return true
		}
	}
	return false
}

func newProvenance(modelID string) Provenance {
	return Provenance{
		GeneratedBy:  GeneratedBy,
		ModelID:      modelID,
		GenerationTS: time.Now().UTC().Format(time.RFC3339),
	}
}

func standardsDisclaimer(code *string) *string {
	if code == nil || strings.TrimSpace(*code) == "" {
		return nil
	}
	msg := "Standards alignment is indicative; verify against your district framework before publishing."
	return &msg
}

// Generate runs parallel component generation and returns the updated package.
func Generate(ctx context.Context, client ChatClient, input InputParams, pkg PackageResult, opts GenerateOptions) PackageResult {
	if err := ValidateInput(input); err != nil {
		pkg.Status = "failed"
		return pkg
	}
	input = RedactInput(input)
	if pkg.StandardsDisclaimer == nil {
		pkg.StandardsDisclaimer = standardsDisclaimer(input.StandardsCode)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	run := func(key string, fn func() (json.RawMessage, error)) {
		if !shouldGenerate(key, opts.OnlyKeys) {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			setComponentStatus(&pkg, key, StatusProcessing, nil, nil, nil)
			mu.Unlock()

			if forced, ok := opts.ForceFail[key]; ok {
				errMsg := forced.Error()
				mu.Lock()
				setComponentStatus(&pkg, key, StatusFailed, &errMsg, nil, nil)
				mu.Unlock()
				recordComponentFailure(key)
				return
			}

			content, err := fn()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errMsg := err.Error()
				setComponentStatus(&pkg, key, StatusFailed, &errMsg, nil, nil)
				recordComponentFailure(key)
				return
			}
			prov := newProvenance(opts.ModelID)
			setComponentStatus(&pkg, key, StatusCompleted, nil, content, &prov)
		}()
	}

	start := time.Now()
	recordRequest()

	run(ComponentLessonPlan, func() (json.RawMessage, error) {
		md, err := generateLessonPlan(ctx, client, opts.ModelID, opts.Prompts.LessonPlan, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(LessonPlanContent{Markdown: md})
	})

	run(ComponentQuiz, func() (json.RawMessage, error) {
		qs, err := generateQuiz(ctx, client, opts.ModelID, opts.Prompts.Quiz, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(QuizContent{Questions: qs})
	})

	run(ComponentRubric, func() (json.RawMessage, error) {
		r, err := generateRubric(ctx, client, opts.ModelID, opts.Prompts.Rubric, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(RubricContent{Rubric: *r})
	})

	levels := input.DifferentiationLevels
	if len(levels) == 0 {
		levels = []string{"on_grade"}
	}
	for _, lvl := range levels {
		level := lvl
		key := ComponentActivityPrefix + level
		run(key, func() (json.RawMessage, error) {
			md, err := generateActivity(ctx, client, opts.ModelID, opts.Prompts.Activity, input, level)
			if err != nil {
				return nil, err
			}
			return json.Marshal(ActivityContent{Level: level, Markdown: md})
		})
	}

	wg.Wait()
	recordLatency(time.Since(start))

	pkg.Status = "completed"
	for i := range pkg.Components {
		if pkg.Components[i].Status == StatusFailed {
			// Partial success is allowed (NFR Reliability).
			continue
		}
	}
	return pkg
}

func setComponentStatus(pkg *PackageResult, key string, status ComponentStatus, errMsg *string, content json.RawMessage, prov *Provenance) {
	for i := range pkg.Components {
		if pkg.Components[i].Key != key {
			continue
		}
		pkg.Components[i].Status = status
		pkg.Components[i].Error = errMsg
		if content != nil {
			pkg.Components[i].Content = content
		}
		if prov != nil {
			pkg.Components[i].Provenance = prov
		}
		return
	}
	pkg.Components = append(pkg.Components, ComponentSlot{
		Key: key, Status: status, Error: errMsg, Content: content, Provenance: prov,
	})
}

func generateLessonPlan(ctx context.Context, client ChatClient, model, sysPrompt string, input InputParams) (string, error) {
	_ = ctx
	user := buildContextPrompt(input) + "\n\nWrite a complete lesson plan for this objective."
	text, err := chatMarkdown(client, model, sysPrompt, user)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func generateActivity(ctx context.Context, client ChatClient, model, sysPrompt string, input InputParams, level string) (string, error) {
	_ = ctx
	user := buildContextPrompt(input) + fmt.Sprintf("\n\nDifferentiation level: %s\nWrite one differentiated activity.", level)
	text, err := chatMarkdown(client, model, sysPrompt, user)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func generateQuiz(ctx context.Context, client ChatClient, model, sysPrompt string, input InputParams) ([]coursemodulequiz.QuizQuestion, error) {
	_ = ctx
	user := buildContextPrompt(input) + "\n\nGenerate exactly 5 formative assessment questions as an exit ticket aligned to the objective."
	text, err := chatJSON(client, model, sysPrompt, user)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Questions []coursemodulequiz.QuizQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse quiz JSON: %w", err)
	}
	if len(payload.Questions) == 0 {
		return nil, fmt.Errorf("quiz generation returned no questions")
	}
	for i := range payload.Questions {
		if payload.Questions[i].ID == "" {
			payload.Questions[i].ID = uuid.New().String()
		}
		if payload.Questions[i].QuestionType == "" {
			payload.Questions[i].QuestionType = "short_answer"
		}
		if payload.Questions[i].Points == 0 {
			payload.Questions[i].Points = 1
		}
		if payload.Questions[i].EstimatedMinutes == 0 {
			payload.Questions[i].EstimatedMinutes = 2
		}
		payload.Questions[i].Required = true
	}
	return payload.Questions, nil
}

func generateRubric(ctx context.Context, client ChatClient, model, sysPrompt string, input InputParams) (*assignmentrubric.RubricDefinition, error) {
	_ = ctx
	user := buildContextPrompt(input) + "\n\nCreate a rubric for the open-ended portion of this lesson."
	text, err := chatJSON(client, model, sysPrompt, user)
	if err != nil {
		return nil, err
	}
	var rubric assignmentrubric.RubricDefinition
	if err := json.Unmarshal([]byte(text), &rubric); err != nil {
		return nil, fmt.Errorf("parse rubric JSON: %w", err)
	}
	for i := range rubric.Criteria {
		if rubric.Criteria[i].ID == uuid.Nil {
			rubric.Criteria[i].ID = uuid.New()
		}
	}
	if err := assignmentrubric.ValidateRubricDefinition(&rubric); err != nil {
		return nil, err
	}
	return &rubric, nil
}

func buildContextPrompt(input InputParams) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Learning objective: %s\n", input.LearningObjective)
	fmt.Fprintf(&b, "Grade level: %s\n", input.GradeLevel)
	fmt.Fprintf(&b, "Subject: %s\n", input.Subject)
	if input.DurationMinutes != nil {
		fmt.Fprintf(&b, "Lesson duration (minutes): %d\n", *input.DurationMinutes)
	}
	if input.StandardsCode != nil && strings.TrimSpace(*input.StandardsCode) != "" {
		fmt.Fprintf(&b, "Standards code (align when possible): %s\n", strings.TrimSpace(*input.StandardsCode))
	}
	return b.String()
}

func chatMarkdown(client ChatClient, model, sysPrompt, user string) (string, error) {
	res, err := client.ChatCompletion(model, []openrouter.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: user},
	})
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

func chatJSON(client ChatClient, model, sysPrompt, user string) (string, error) {
	res, err := client.ChatCompletion(model, []openrouter.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: user},
	}, openrouter.ChatOptions{JSONMode: true})
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(res.Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text), nil
}

// RegenerateComponent re-runs a single component and merges it into the package.
func RegenerateComponent(ctx context.Context, client ChatClient, input InputParams, pkg PackageResult, componentKey string, opts GenerateOptions) (PackageResult, error) {
	opts.OnlyKeys = []string{componentKey}
	return Generate(ctx, client, input, pkg, opts), nil
}

// MarshalPackage serializes a package for DB storage.
func MarshalPackage(pkg PackageResult) (json.RawMessage, error) {
	return json.Marshal(pkg)
}

// UnmarshalPackage deserializes stored job result.
func UnmarshalPackage(raw json.RawMessage) (PackageResult, error) {
	var pkg PackageResult
	if len(raw) == 0 {
		return pkg, nil
	}
	err := json.Unmarshal(raw, &pkg)
	return pkg, err
}

// DefaultQuizPrompt is used when the system_prompts row is missing.
const DefaultQuizPrompt = `You generate quiz questions for an LMS. Respond with ONLY valid JSON.
The JSON must be {"questions":[...]} using camelCase keys: prompt, questionType, choices, correctChoiceIndex, points, estimatedMinutes.
Generate exactly 5 questions aligned to the learning objective.`

// DefaultLessonPlanPrompt fallback.
const DefaultLessonPlanPrompt = `You are an expert curriculum designer. Write a lesson plan in Markdown for the given objective, grade, and subject.`

// DefaultActivityPrompt fallback.
const DefaultActivityPrompt = `You are an expert curriculum designer. Write a differentiated activity in Markdown.`

// DefaultRubricPrompt fallback.
const DefaultRubricPrompt = `You generate assignment rubrics. Respond with ONLY valid JSON matching the rubric schema with criteria and levels.`
