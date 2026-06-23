// Package gradingagent scores student submissions via OpenRouter using instructor-authored prompts.
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
	"github.com/lextures/lextures/server/internal/service/aitutor"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/openrouter"
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
}

// Service scores submissions using OpenRouter.
type Service struct {
	Client    *openrouter.Client
	Storage   filestorage.Driver
	FilesRoot string
	Pool      *pgxpool.Pool
}

// RunPrompt executes an AI workflow node prompt with a fixed system prompt and optional JSON mode.
func (s *Service) RunPrompt(ctx context.Context, modelID, systemPrompt, prompt, input string, jsonMode bool) (string, int, int, error) {
	if s.Client == nil {
		return "", 0, 0, fmt.Errorf("AI provider not configured")
	}
	model := strings.TrimSpace(modelID)
	if model == "" {
		return "", 0, 0, fmt.Errorf("grader agent model not configured")
	}
	systemPrompt = strings.TrimSpace(systemPrompt)
	prompt = strings.TrimSpace(prompt)
	input = strings.TrimSpace(input)
	messages := make([]openrouter.Message, 0, 3)
	if systemPrompt != "" {
		messages = append(messages, openrouter.Message{Role: "system", Content: systemPrompt})
	}
	if prompt != "" {
		messages = append(messages, openrouter.Message{Role: "user", Content: prompt})
	}
	if input != "" {
		messages = append(messages, openrouter.Message{Role: "user", Content: input})
	}
	var chat openrouter.ChatResult
	var err error
	if jsonMode {
		chat, err = s.Client.ChatCompletion(model, messages, openrouter.ChatOptions{JSONMode: true})
		if err != nil && (strings.Contains(err.Error(), "response_format") || strings.Contains(err.Error(), "status 400")) {
			chat, err = s.Client.ChatCompletion(model, messages)
		}
	} else {
		chat, err = s.Client.ChatCompletion(model, messages)
	}
	if err != nil {
		return "", 0, 0, err
	}
	text := strings.TrimSpace(chat.Text)
	if text == "" {
		return "", chat.Usage.PromptTokens, chat.Usage.CompletionTokens, fmt.Errorf("openrouter: empty model response")
	}
	return text, chat.Usage.PromptTokens, chat.Usage.CompletionTokens, nil
}

// Score runs the grading agent against one submission.
func (s *Service) Score(ctx context.Context, req ScoreRequest) (ScoreResult, error) {
	if s.Client == nil {
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
	chat, err := s.Client.ChatCompletion(model, messages, openrouter.ChatOptions{JSONMode: true})
	if err != nil {
		// Some models reject json_object; retry once without structured output.
		if strings.Contains(err.Error(), "response_format") || strings.Contains(err.Error(), "status 400") {
			chat, err = s.Client.ChatCompletion(model, messages)
		}
		if err != nil {
			return ScoreResult{}, err
		}
	}
	if strings.TrimSpace(chat.Text) == "" {
		return ScoreResult{}, fmt.Errorf("openrouter: empty model response")
	}
	out, err := ParseAndClampModelOutput(chat.Text, req.Rubric, req.MaxPoints)
	if err != nil {
		return ScoreResult{}, err
	}
	return ScoreResult{
		Output:           out,
		ModelID:          model,
		PromptTokens:     chat.Usage.PromptTokens,
		CompletionTokens: chat.Usage.CompletionTokens,
		CostUSD:          chat.Usage.CostUSD,
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

