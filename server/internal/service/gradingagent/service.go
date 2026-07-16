// Package gradingagent scores student submissions via a provider-agnostic AI
// backend (server/internal/service/aiprovider) using instructor-authored prompts.
package gradingagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/aitutor"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// ScoreRequest holds inputs for one grading call.
type ScoreRequest struct {
	InstructorPrompt         string
	IncludeAssignmentContent bool
	IncludeRubric            bool
	ModelID                  string
	AssignmentMarkdown       string
	Rubric                   *assignmentrubric.RubricDefinition
	MaxPoints                float64
	SubmissionText           string
}

// ScoreResult holds the model output and usage metadata.
type ScoreResult struct {
	Output           GradeOutput
	ModelID          string
	PromptTokens     int
	CompletionTokens int
	CostUSD          float64
	// CallMeta describes which AI provider/model actually answered the call
	// (AP.4 FR-4); prefer this over ModelID alone for usage logging.
	CallMeta aiprovider.CallMeta
}

// Service scores submissions using a provider-agnostic AI backend (AP.4).
type Service struct {
	// AI is the org-scoped text completion surface; required for RunPrompt/Score.
	AI aiprovider.ScopedCompleter
	// Vision is the org-scoped vision completion surface; required only for
	// ScoreWithVision. Callers may pass the same value as AI when it also
	// implements aiprovider.ScopedVisionCompleter (e.g. aiprovider.BoundCompleter).
	Vision    aiprovider.ScopedVisionCompleter
	Storage   filestorage.Driver
	FilesRoot string
	Pool      *pgxpool.Pool
	// LastMeta records provider/model metadata from the most recent RunPrompt or
	// RunBuilderPrompt call, for callers that need usage logging without
	// threading CallMeta through those return signatures. Score and
	// ScoreWithVision return CallMeta on ScoreResult instead.
	LastMeta aiprovider.CallMeta
}

// isJSONModeRetryable reports whether an error indicates the provider rejected
// structured JSON output and the call should be retried without it.
func isJSONModeRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "response_format") || strings.Contains(msg, "status 400")
}

// RunPrompt executes an AI workflow node prompt with a fixed system prompt and optional JSON mode.
func (s *Service) RunPrompt(ctx context.Context, modelID, systemPrompt, prompt, input string, jsonMode bool) (string, int, int, float64, error) {
	if s.AI == nil {
		return "", 0, 0, 0, fmt.Errorf("AI provider not configured")
	}
	model := strings.TrimSpace(modelID)
	if model == "" {
		return "", 0, 0, 0, fmt.Errorf("grader agent model not configured")
	}
	systemPrompt = strings.TrimSpace(systemPrompt)
	prompt = strings.TrimSpace(prompt)
	input = strings.TrimSpace(input)
	messages := make([]aiprovider.Message, 0, 3)
	if systemPrompt != "" {
		messages = append(messages, aiprovider.Message{Role: "system", Content: systemPrompt})
	}
	if prompt != "" {
		messages = append(messages, aiprovider.Message{Role: "user", Content: prompt})
	}
	if input != "" {
		messages = append(messages, aiprovider.Message{Role: "user", Content: input})
	}
	var chat aiprovider.ChatResult
	var meta aiprovider.CallMeta
	var err error
	if jsonMode {
		chat, meta, err = s.AI.Complete(ctx, model, messages, aiprovider.ChatOptions{JSONMode: true})
		if err != nil && isJSONModeRetryable(err) {
			chat, meta, err = s.AI.Complete(ctx, model, messages)
		}
	} else {
		chat, meta, err = s.AI.Complete(ctx, model, messages)
	}
	s.LastMeta = meta
	if err != nil {
		return "", 0, 0, 0, err
	}
	text := strings.TrimSpace(chat.Text)
	if text == "" {
		return "", chat.Usage.PromptTokens, chat.Usage.CompletionTokens, chat.Usage.CostUSD, fmt.Errorf("empty model response")
	}
	return text, chat.Usage.PromptTokens, chat.Usage.CompletionTokens, chat.Usage.CostUSD, nil
}

// RunBuilderPrompt runs a single structured-JSON generation with a bounded output
// length. Used by the AI workflow builder, where a slow/large model could otherwise
// run past the client timeout. Honors the same JSON-mode fallback as RunPrompt.
func (s *Service) RunBuilderPrompt(ctx context.Context, modelID, systemPrompt, prompt, input string, maxTokens int) (string, int, int, float64, error) {
	if s.AI == nil {
		return "", 0, 0, 0, fmt.Errorf("AI provider not configured")
	}
	model := strings.TrimSpace(modelID)
	if model == "" {
		return "", 0, 0, 0, fmt.Errorf("grader agent model not configured")
	}
	messages := make([]aiprovider.Message, 0, 3)
	if s := strings.TrimSpace(systemPrompt); s != "" {
		messages = append(messages, aiprovider.Message{Role: "system", Content: s})
	}
	if p := strings.TrimSpace(prompt); p != "" {
		messages = append(messages, aiprovider.Message{Role: "user", Content: p})
	}
	if i := strings.TrimSpace(input); i != "" {
		messages = append(messages, aiprovider.Message{Role: "user", Content: i})
	}
	chat, meta, err := s.AI.Complete(ctx, model, messages, aiprovider.ChatOptions{JSONMode: true, MaxTokens: maxTokens})
	if err != nil && isJSONModeRetryable(err) {
		chat, meta, err = s.AI.Complete(ctx, model, messages, aiprovider.ChatOptions{MaxTokens: maxTokens})
	}
	s.LastMeta = meta
	if err != nil {
		return "", 0, 0, 0, err
	}
	text := strings.TrimSpace(chat.Text)
	if text == "" {
		return "", chat.Usage.PromptTokens, chat.Usage.CompletionTokens, chat.Usage.CostUSD, fmt.Errorf("empty model response")
	}
	return text, chat.Usage.PromptTokens, chat.Usage.CompletionTokens, chat.Usage.CostUSD, nil
}

// ScoreWithVision grades a submission from image/PDF pages using a vision-capable model.
func (s *Service) ScoreWithVision(ctx context.Context, req ScoreRequest, imageDataURLs []string) (ScoreResult, error) {
	if s.Vision == nil {
		return ScoreResult{}, fmt.Errorf("AI provider not configured")
	}
	model := strings.TrimSpace(req.ModelID)
	if model == "" {
		return ScoreResult{}, fmt.Errorf("grader agent model not configured")
	}
	if len(imageDataURLs) == 0 {
		return ScoreResult{}, fmt.Errorf("no submission images available")
	}
	messages := BuildMessages(
		req.InstructorPrompt,
		req.IncludeAssignmentContent,
		req.IncludeRubric,
		req.AssignmentMarkdown,
		req.Rubric,
		"[Submission provided as attached image(s) or scanned document page(s). Read and grade the visual content.]",
		req.MaxPoints,
	)
	systemPrompt := ""
	userText := ""
	if len(messages) >= 1 {
		systemPrompt = messages[0].Content
	}
	if len(messages) >= 2 {
		userText = messages[1].Content
	}
	visionMessages := aiprovider.VisionMessages(systemPrompt, userText, imageDataURLs)
	chat, meta, err := s.Vision.CompleteVision(ctx, model, visionMessages, aiprovider.ChatOptions{JSONMode: true})
	if err != nil && isJSONModeRetryable(err) {
		chat, meta, err = s.Vision.CompleteVision(ctx, model, visionMessages)
	}
	if err != nil {
		return ScoreResult{CallMeta: meta}, err
	}
	if strings.TrimSpace(chat.Text) == "" {
		return ScoreResult{CallMeta: meta}, fmt.Errorf("empty model response")
	}
	out, err := ParseAndClampModelOutput(chat.Text, req.Rubric, req.MaxPoints)
	if err != nil {
		return ScoreResult{CallMeta: meta}, err
	}
	return ScoreResult{
		Output:           out,
		ModelID:          model,
		PromptTokens:     chat.Usage.PromptTokens,
		CompletionTokens: chat.Usage.CompletionTokens,
		CostUSD:          chat.Usage.CostUSD,
		CallMeta:         meta,
	}, nil
}

// Score runs the grading agent against one submission.
func (s *Service) Score(ctx context.Context, req ScoreRequest) (ScoreResult, error) {
	if s.AI == nil {
		return ScoreResult{}, fmt.Errorf("AI provider not configured")
	}
	model := strings.TrimSpace(req.ModelID)
	if model == "" {
		return ScoreResult{}, fmt.Errorf("grader agent model not configured")
	}
	submissionText := aitutor.RedactPII(strings.TrimSpace(req.SubmissionText))
	if submissionText == "" {
		return ScoreResult{}, fmt.Errorf("submission text is empty")
	}
	messages := BuildMessages(
		req.InstructorPrompt,
		req.IncludeAssignmentContent,
		req.IncludeRubric,
		req.AssignmentMarkdown,
		req.Rubric,
		submissionText,
		req.MaxPoints,
	)
	chat, meta, err := s.AI.Complete(ctx, model, messages, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		// Some models reject json_object; retry once without structured output.
		if isJSONModeRetryable(err) {
			chat, meta, err = s.AI.Complete(ctx, model, messages)
		}
		if err != nil {
			return ScoreResult{CallMeta: meta}, err
		}
	}
	if strings.TrimSpace(chat.Text) == "" {
		return ScoreResult{CallMeta: meta}, fmt.Errorf("empty model response")
	}
	out, err := ParseAndClampModelOutput(chat.Text, req.Rubric, req.MaxPoints)
	if err != nil {
		return ScoreResult{CallMeta: meta}, err
	}
	return ScoreResult{
		Output:           out,
		ModelID:          model,
		PromptTokens:     chat.Usage.PromptTokens,
		CompletionTokens: chat.Usage.CompletionTokens,
		CostUSD:          chat.Usage.CostUSD,
		CallMeta:         meta,
	}, nil
}

// ParseAssignmentRubric unmarshals rubric JSON from an assignment row.
func ParseAssignmentRubric(row *coursemoduleassignments.CourseItemAssignmentRow) (*assignmentrubric.RubricDefinition, error) {
	if row == nil || len(row.RubricJSON) == 0 {
		return nil, nil
	}
	var def assignmentrubric.RubricDefinition
	if err := json.Unmarshal(row.RubricJSON, &def); err != nil {
		return nil, err
	}
	if len(def.Criteria) == 0 {
		return nil, nil
	}
	return &def, nil
}

// MaxPointsFromAssignment returns the assignment point ceiling as a float.
func MaxPointsFromAssignment(row *coursemoduleassignments.CourseItemAssignmentRow) float64 {
	if row == nil || row.PointsWorth == nil || *row.PointsWorth <= 0 {
		return 100
	}
	return float64(*row.PointsWorth)
}

// ContentHashInput builds a stable hash input for AI gateway checks.
func ContentHashInput(prompt, submissionText string) string {
	return prompt + "\n---\n" + submissionText
}

func (s *Service) readSubmissionBlob(ctx context.Context, courseCode string, row *coursefiles.Row) ([]byte, error) {
	if row == nil {
		return nil, fmt.Errorf("submission file not found")
	}
	if s.Storage != nil {
		rc, err := s.Storage.GetObject(ctx, row.StorageKey)
		if err == nil {
			defer func() { _ = rc.Close() }()
			b, readErr := io.ReadAll(rc)
			if readErr == nil {
				return b, nil
			}
		}
	}
	root := strings.TrimSpace(s.FilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	if b, err := os.ReadFile(coursefiles.BlobDiskPath(root, courseCode, row.StorageKey)); err == nil {
		return b, nil
	}
	legacyPath := filepath.Join(root, courseCode, row.StorageKey)
	b, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("read submission file: %w", err)
	}
	return b, nil
}
