package welcomedraft

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const MaxPromptMaterial = 8_000

// Input is course metadata only — never enrollments or learner data (CC.10 §11).
type Input struct {
	CourseTitle       string
	CourseDescription string
	StartDate         string // ISO date or empty
	EndDate           string
	Language          string
}

// Draft is a proposed welcome announcement body (markdown/plain text).
type Draft struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// DefaultSystemPrompt for welcome announcement drafts.
const DefaultSystemPrompt = `You draft a short welcome announcement for an LMS course.
Respond with ONLY valid JSON (no markdown fences):
{"subject":"string","body":"string"}

Rules:
- Body should be 120–400 words, warm and practical, covering: welcome, how to start, where to find the syllabus, how to contact the instructor, and first-week expectations.
- Use the course language when provided; otherwise English.
- Do not invent specific student names, office hour times, or policies not in the input.
- Do not include grades, enrollments, or personal data.
- Body may use light markdown (paragraphs, bullet lists).`

// Generate produces a draft welcome announcement.
func Generate(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt string,
	in Input,
) (Draft, aiprovider.CallMeta, error) {
	title := strings.TrimSpace(in.CourseTitle)
	if title == "" {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("course title is required")
	}
	if looksLikeLearnerField(title) || looksLikeLearnerField(in.CourseDescription) {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("welcome draft rejected: learner data fields are not allowed")
	}

	payload := map[string]any{
		"courseTitle":       title,
		"courseDescription": truncateRunes(strings.TrimSpace(in.CourseDescription), 2000),
		"startDate":         strings.TrimSpace(in.StartDate),
		"endDate":           strings.TrimSpace(in.EndDate),
		"language":          strings.TrimSpace(in.Language),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Draft{}, aiprovider.CallMeta{}, err
	}
	user := "Draft a welcome announcement for this course. Input JSON:\n" + string(encoded)
	if utf8.RuneCountInString(user) > MaxPromptMaterial {
		return Draft{}, aiprovider.CallMeta{}, fmt.Errorf("welcome draft prompt is too long")
	}

	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return Draft{}, meta, err
	}
	draft, err := ParseDraftJSON(res.Text)
	if err != nil {
		return Draft{}, meta, err
	}
	return draft, meta, nil
}

// ParseDraftJSON parses model output into a Draft.
func ParseDraftJSON(raw string) (Draft, error) {
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```json"); idx != -1 {
		text = text[idx+7:]
		if endIdx := strings.Index(text, "```"); endIdx != -1 {
			text = text[:endIdx]
		}
	}
	text = strings.TrimSpace(text)
	var d Draft
	if err := json.Unmarshal([]byte(text), &d); err != nil {
		return Draft{}, fmt.Errorf("parse welcome draft JSON: %w", err)
	}
	d.Subject = strings.TrimSpace(d.Subject)
	d.Body = strings.TrimSpace(d.Body)
	if d.Body == "" {
		return Draft{}, fmt.Errorf("parse welcome draft JSON: empty body")
	}
	if d.Subject == "" {
		d.Subject = "Welcome"
	}
	return d, nil
}

func looksLikeLearnerField(s string) bool {
	lower := strings.ToLower(s)
	for _, banned := range []string{"student name", "submission", "accommodation", "enrollment list"} {
		if strings.Contains(lower, banned) {
			return true
		}
	}
	return false
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}
