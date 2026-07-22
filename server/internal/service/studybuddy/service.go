// Package studybuddy implements the homeschool AI study buddy (plan 15.12).
package studybuddy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/learnergoals"
	studybuddyrepo "github.com/lextures/lextures/server/internal/repos/studybuddy"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/aitutor"
	"github.com/lextures/lextures/server/internal/service/notebookrag"
)

const (
	maxMessageChars     = 2000
	maxHistoryTurns     = 20
	maxRAGChunks        = 5
	quizStrugglePct     = 40.0
	staleModuleDays     = 5
	maxStruggleConcepts = 8
)

// Prompt is a proactive study buddy nudge shown in the UI.
type Prompt struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
	ItemID  string `json:"itemId,omitempty"`
}

// MemorySummary is the learner-visible memory context.
type MemorySummary struct {
	GoalsSummary       *string  `json:"goalsSummary,omitempty"`
	StruggleConcepts   []string `json:"struggleConcepts"`
	LastSessionSummary *string  `json:"lastSessionSummary,omitempty"`
	LastActiveAt       *string  `json:"lastActiveAt,omitempty"`
}

// Citation is a source link returned with assistant messages.
type Citation struct {
	ItemID  string `json:"itemId"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
}

// MessageResult is the non-streaming payload shape (streaming uses SSE events).
type MessageResult struct {
	SessionID string     `json:"sessionId"`
	Text      string     `json:"text"`
	Citations []Citation `json:"citations"`
}

// Service orchestrates study buddy memory, prompts, and LLM calls.
type Service struct {
	Pool   *pgxpool.Pool
	Config config.Config
}

func (s *Service) enabled() bool {
	return s.Config.FFAIStudyBuddy && s.Pool != nil
}

// RefreshMemory syncs goals and struggle concepts from learner data sources.
func (s *Service) RefreshMemory(ctx context.Context, userID, courseID uuid.UUID) (*studybuddyrepo.MemoryRow, error) {
	if !s.enabled() {
		return nil, nil
	}
	var goalsSummary *string
	if goals, err := learnergoals.Get(ctx, s.Pool, userID); err == nil && goals != nil {
		parts := make([]string, 0, 2)
		if strings.TrimSpace(goals.Topic) != "" {
			parts = append(parts, "Topic: "+strings.TrimSpace(goals.Topic))
		}
		if goals.GoalText != nil && strings.TrimSpace(*goals.GoalText) != "" {
			parts = append(parts, "Goal: "+strings.TrimSpace(*goals.GoalText))
		}
		if len(parts) > 0 {
			joined := strings.Join(parts, ". ")
			goalsSummary = &joined
		}
	}
	struggles, err := studybuddyrepo.LowQuizStruggles(ctx, s.Pool, courseID, userID, quizStrugglePct, maxStruggleConcepts)
	if err != nil {
		return nil, err
	}
	existing, err := studybuddyrepo.GetMemory(ctx, s.Pool, userID, courseID)
	if err != nil {
		return nil, err
	}
	var sessionSummary *string
	if existing != nil {
		sessionSummary = existing.LastSessionSummary
	}
	return studybuddyrepo.UpsertMemory(ctx, s.Pool, userID, courseID, goalsSummary, struggles, sessionSummary)
}

// GetMemorySummary returns memory for the learner, refreshing struggle data first.
func (s *Service) GetMemorySummary(ctx context.Context, userID, courseID uuid.UUID) (*MemorySummary, error) {
	row, err := s.RefreshMemory(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &MemorySummary{StruggleConcepts: []string{}}, nil
	}
	out := &MemorySummary{
		GoalsSummary:       row.GoalsSummary,
		StruggleConcepts:   row.StruggleConcepts,
		LastSessionSummary: row.LastSessionSummary,
	}
	if row.LastActiveAt != nil {
		formatted := row.LastActiveAt.UTC().Format(time.RFC3339)
		out.LastActiveAt = &formatted
	}
	return out, nil
}

// ClearMemory deletes persisted memory (GDPR erasure).
func (s *Service) ClearMemory(ctx context.Context, userID, courseID uuid.UUID) error {
	return studybuddyrepo.DeleteMemory(ctx, s.Pool, userID, courseID)
}

// ListPrompts returns proactive study nudges for the learner.
func (s *Service) ListPrompts(ctx context.Context, userID, courseID uuid.UUID, now time.Time) ([]Prompt, error) {
	if !s.enabled() {
		return nil, nil
	}
	memory, err := s.RefreshMemory(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	var prompts []Prompt
	for _, concept := range memory.StruggleConcepts {
		prompts = append(prompts, Prompt{
			ID:      "struggle-" + slug(concept),
			Kind:    "quiz_struggle",
			Message: fmt.Sprintf("You struggled with %s last time — want to review?", concept),
		})
	}
	eid, err := enrollment.GetStudentEnrollmentID(ctx, s.Pool, courseID, userID)
	if err != nil {
		return prompts, err
	}
	if eid != nil {
		stale, err := studybuddyrepo.StaleVisitedModules(ctx, s.Pool, *eid, courseID, staleModuleDays, 3)
		if err != nil {
			return prompts, err
		}
		for _, mod := range stale {
			prompts = append(prompts, Prompt{
				ID:      "stale-" + slug(mod),
				Kind:    "stale_content",
				Message: fmt.Sprintf("You haven't reviewed %s in %d days — want a quick quiz?", mod, staleModuleDays),
			})
		}
	}
	if s.Config.SRSPracticeEnabled {
		prompts = appendReviewDuePrompt(prompts, now)
	}
	return prompts, nil
}

func appendReviewDuePrompt(prompts []Prompt, now time.Time) []Prompt {
	_ = now
	// SRS review queue persistence is not yet wired; placeholder for plan 1.5 integration.
	return prompts
}

// BuildMessages assembles the provider-agnostic message list for a study buddy turn.
func (s *Service) BuildMessages(
	courseTitle string,
	memory *studybuddyrepo.MemoryRow,
	priorLevel string,
	displayName string,
	history []studybuddyrepo.Message,
	userMessage string,
	ragContext string,
	hasRAG bool,
) []aiprovider.Message {
	sys := BuildSystemPrompt(courseTitle, memory, priorLevel, displayName, hasRAG)
	msgs := []aiprovider.Message{{Role: "system", Content: sys}}
	start := 0
	if len(history) > maxHistoryTurns*2 {
		start = len(history) - maxHistoryTurns*2
	}
	for _, m := range history[start:] {
		msgs = append(msgs, aiprovider.Message{Role: m.Role, Content: m.Content})
	}
	body := userMessage
	if strings.TrimSpace(ragContext) != "" {
		body = fmt.Sprintf("Student question:\n---\n%s\n---\n\nRelevant course material excerpts (only use these as evidence):\n%s", userMessage, ragContext)
	}
	msgs = append(msgs, aiprovider.Message{Role: "user", Content: body})
	return msgs
}

// RetrieveCourseRAG loads course pages and returns top chunks + citations.
func (s *Service) RetrieveCourseRAG(ctx context.Context, courseID uuid.UUID, courseCode, courseTitle, question string) (string, []Citation, bool) {
	pages, err := studybuddyrepo.ListCourseContentPages(ctx, s.Pool, courseID)
	if err != nil || len(pages) == 0 {
		return "", nil, false
	}
	docs := make([]notebookrag.DocInput, 0, len(pages))
	itemByTitle := make(map[string]uuid.UUID, len(pages))
	for _, p := range pages {
		label := fmt.Sprintf("%s (%s)", p.Title, courseCode)
		docs = append(docs, notebookrag.DocInput{
			CourseCode:  courseCode,
			CourseTitle: label,
			Markdown:    p.Body,
		})
		itemByTitle[strings.ToLower(p.Title)] = p.ItemID
	}
	sources := notebookrag.RetrieveTopSources(question, docs, maxRAGChunks)
	if len(sources) == 0 {
		return "", nil, false
	}
	var b strings.Builder
	citations := make([]Citation, 0, len(sources))
	for i, src := range sources {
		b.WriteString("\n\n--- Excerpt ")
		fmt.Fprintf(&b, "%d", i+1)
		b.WriteString(" — ")
		b.WriteString(src.CourseTitle)
		b.WriteString(" ---\n")
		b.WriteString(src.Excerpt)
		title := strings.TrimSpace(strings.Split(src.CourseTitle, " (")[0])
		itemID := ""
		if id, ok := itemByTitle[strings.ToLower(title)]; ok {
			itemID = id.String()
		}
		citations = append(citations, Citation{
			ItemID:  itemID,
			Title:   title,
			Excerpt: src.Excerpt,
		})
	}
	return b.String(), citations, true
}

// ValidateMessage cleans and validates a learner message.
func ValidateMessage(raw string) (string, error) {
	if len([]rune(raw)) > maxMessageChars {
		return "", fmt.Errorf("message too long (max %d characters)", maxMessageChars)
	}
	cleaned := strings.TrimSpace(aitutor.RedactPII(raw))
	if cleaned == "" {
		return "", fmt.Errorf("message cannot be empty")
	}
	return cleaned, nil
}

// SummarizeSession builds a short rolling summary from recent turns.
func SummarizeSession(turns []studybuddyrepo.Message) string {
	if len(turns) == 0 {
		return ""
	}
	start := 0
	if len(turns) > 6 {
		start = len(turns) - 6
	}
	var parts []string
	for _, m := range turns[start:] {
		snippet := m.Content
		if len(snippet) > 120 {
			snippet = snippet[:117] + "…"
		}
		parts = append(parts, m.Role+": "+snippet)
	}
	summary := strings.Join(parts, " ")
	if len(summary) > 500 {
		summary = summary[:497] + "…"
	}
	return summary
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			b.WriteRune(ch)
		}
	}
	out := b.String()
	if out == "" {
		return "item"
	}
	return out
}
