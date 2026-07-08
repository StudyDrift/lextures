// Package tutorsession implements persistent AI tutor sessions with RAG citations (plan 19.1).
package tutorsession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	studybuddyrepo "github.com/lextures/lextures/server/internal/repos/studybuddy"
	tutorrepo "github.com/lextures/lextures/server/internal/repos/tutorsession"
	"github.com/lextures/lextures/server/internal/service/aitutor"
	"github.com/lextures/lextures/server/internal/service/notebookrag"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const (
	maxMessageChars     = 2000
	HistoryMessageLimit = 10
	maxRAGChunks        = 5
	disclosureText      = "I am an AI tutor. I can make mistakes — please verify important information with your instructor."
)

// Service orchestrates persistent tutor session logic.
type Service struct {
	Pool *pgxpool.Pool
}

// ConceptRef is a course concept used for tagging confusion signals.
type ConceptRef struct {
	ID   uuid.UUID
	Name string
}

// BuildSystemPrompt returns the tutor system prompt for a course.
func BuildSystemPrompt(courseTitle string, hasRAG bool) string {
	base := aitutor.BuildSystemPrompt(courseTitle)
	if hasRAG {
		base += "\n- Ground answers in the provided course material excerpts. Include citations in your response when referencing course content."
	} else {
		base += "\n- Course material retrieval failed; answer from general knowledge but prominently note that the response is not grounded in course materials."
	}
	return base
}

// ValidateMessage redacts PII and validates length.
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

// RetrieveCourseRAG loads course pages and returns context + citations.
func (s *Service) RetrieveCourseRAG(ctx context.Context, courseID uuid.UUID, courseCode, courseTitle, question string) (string, []tutorrepo.Citation, bool) {
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
	citations := make([]tutorrepo.Citation, 0, len(sources))
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
		chunkID := chunkIDFromExcerpt(src.Excerpt)
		citations = append(citations, tutorrepo.Citation{
			SourceID: itemID,
			ChunkID:  chunkID,
			Excerpt:  src.Excerpt,
			Title:    title,
		})
	}
	return b.String(), citations, true
}

func chunkIDFromExcerpt(excerpt string) string {
	sum := sha256.Sum256([]byte(excerpt))
	return hex.EncodeToString(sum[:8])
}

// FilterValidCitations keeps only citations whose source_id exists in the retrieved set.
func FilterValidCitations(citations []tutorrepo.Citation, retrieved []tutorrepo.Citation) []tutorrepo.Citation {
	if len(citations) == 0 {
		return retrieved
	}
	valid := make(map[string]struct{}, len(retrieved))
	for _, c := range retrieved {
		if c.SourceID != "" {
			valid[c.SourceID] = struct{}{}
		}
	}
	out := make([]tutorrepo.Citation, 0, len(citations))
	for _, c := range citations {
		if c.SourceID == "" {
			continue
		}
		if _, ok := valid[c.SourceID]; ok {
			out = append(out, c)
		}
	}
	if len(out) == 0 && len(retrieved) > 0 {
		return retrieved[:1]
	}
	return out
}

// BuildMessages assembles OpenRouter messages for a tutor turn.
func BuildMessages(
	courseTitle string,
	history []tutorrepo.Message,
	userMessage, ragContext string,
	hasRAG bool,
	profileScaffolding string,
) []openrouter.Message {
	sys := BuildSystemPrompt(courseTitle, hasRAG)
	if strings.TrimSpace(profileScaffolding) != "" {
		sys += profileScaffolding
	}
	msgs := []openrouter.Message{{Role: "system", Content: sys}}
	start := 0
	if len(history) > HistoryMessageLimit {
		start = len(history) - HistoryMessageLimit
	}
	for _, m := range history[start:] {
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, openrouter.Message{Role: m.Role, Content: m.Content})
	}
	body := userMessage
	if strings.TrimSpace(ragContext) != "" {
		body = fmt.Sprintf("Student question:\n---\n%s\n---\n\nRelevant course material excerpts (only use these as evidence):\n%s", userMessage, ragContext)
	}
	msgs = append(msgs, openrouter.Message{Role: "user", Content: body})
	return msgs
}

// DetectConceptTags matches concept names mentioned in the student message.
func DetectConceptTags(message string, concepts []ConceptRef) []uuid.UUID {
	lower := strings.ToLower(message)
	var ids []uuid.UUID
	for _, c := range concepts {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(name)) {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// ListCourseConcepts returns concepts for a course.
func (s *Service) ListCourseConcepts(ctx context.Context, courseID uuid.UUID) ([]ConceptRef, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT id, name FROM course.concepts WHERE course_id = $1 ORDER BY name ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConceptRef
	for rows.Next() {
		var c ConceptRef
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// EnsureDisclosure inserts the system disclosure message when missing.
func (s *Service) EnsureDisclosure(ctx context.Context, sessionID uuid.UUID) error {
	has, err := tutorrepo.HasSystemDisclosure(ctx, s.Pool, sessionID)
	if err != nil || has {
		return err
	}
	_, err = tutorrepo.AppendMessage(ctx, s.Pool, sessionID, "system", disclosureText, nil, nil, 0)
	return err
}

// SessionTitleFromMessage derives a short session title from the first user message.
func SessionTitleFromMessage(message string) string {
	t := strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if utf8.RuneCountInString(t) <= 48 {
		return t
	}
	rs := []rune(t)
	return string(rs[:45]) + "…"
}

// EstimateTokens provides a rough token estimate for budgeting.
func EstimateTokens(text string) int {
	n := len(text)/4 + 1
	if n < 1 {
		return 1
	}
	return n
}

// ConfusionSince returns the time window for instructor concept confusion digests.
func ConfusionSince(now time.Time) time.Time {
	return now.Add(-7 * 24 * time.Hour)
}

// DisclosureMessage returns the first-use disclosure text.
func DisclosureMessage() string {
	return disclosureText
}
