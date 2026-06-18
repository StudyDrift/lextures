package gradecomment

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Comment is one message in a submission grade feedback thread.
type Comment struct {
	ID          string  `json:"id"`
	UserID      *string `json:"userId,omitempty"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	Body        string  `json:"body"`
	CreatedAt   string  `json:"createdAt"`
	Source      string  `json:"source,omitempty"`
}

const maxStoredComments = 200

// ParseList decodes stored JSON into comments. Nil/empty input yields an empty slice.
func ParseList(raw []byte) ([]Comment, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []Comment
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MarshalList encodes comments for storage.
func MarshalList(comments []Comment) ([]byte, error) {
	if len(comments) == 0 {
		return nil, nil
	}
	return json.Marshal(comments)
}

// Flatten renders comments as legacy instructor_comment text.
func Flatten(comments []Comment) string {
	if len(comments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(comments))
	for _, c := range comments {
		body := strings.TrimSpace(c.Body)
		if body == "" {
			continue
		}
		name := strings.TrimSpace(c.DisplayName)
		if name != "" && name != "Comment" && name != "Rubric" {
			parts = append(parts, name+": "+body)
		} else {
			parts = append(parts, body)
		}
	}
	return strings.Join(parts, "\n\n")
}

// LatestBody returns the most recent non-empty comment body.
func LatestBody(comments []Comment) *string {
	for i := len(comments) - 1; i >= 0; i-- {
		t := strings.TrimSpace(comments[i].Body)
		if t != "" {
			return &t
		}
	}
	return nil
}

// ParseLegacyFlat converts pre-JSON instructor_comment text into comments.
func ParseLegacyFlat(text string) []Comment {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	chunks := strings.Split(trimmed, "\n\n")
	out := make([]Comment, 0, len(chunks))
	for i, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		displayName := "Comment"
		body := chunk
		if idx := strings.Index(chunk, ": "); idx > 0 {
			displayName = strings.TrimSpace(chunk[:idx])
			body = strings.TrimSpace(chunk[idx+2:])
		}
		if body == "" {
			continue
		}
		out = append(out, Comment{
			ID:          fmt.Sprintf("legacy-%d", i),
			DisplayName: displayName,
			Body:        body,
			Source:      "legacy",
		})
	}
	return out
}

// ResolveList prefers structured JSON and falls back to legacy flat text.
func ResolveList(commentsJSON []byte, legacyFlat *string) []Comment {
	if parsed, err := ParseList(commentsJSON); err == nil && len(parsed) > 0 {
		return parsed
	}
	if legacyFlat != nil {
		if legacy := ParseLegacyFlat(*legacyFlat); len(legacy) > 0 {
			return legacy
		}
	}
	return nil
}

// Append adds a comment and returns updated slice plus marshaled JSON and flat text.
func Append(existing []Comment, c Comment) ([]Comment, []byte, *string, error) {
	if strings.TrimSpace(c.Body) == "" {
		return existing, nil, nil, fmt.Errorf("empty comment body")
	}
	if strings.TrimSpace(c.ID) == "" {
		c.ID = uuid.New().String()
	}
	if strings.TrimSpace(c.CreatedAt) == "" {
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(c.DisplayName) == "" {
		c.DisplayName = "Comment"
	}
	next := append(append([]Comment{}, existing...), c)
	if len(next) > maxStoredComments {
		next = next[len(next)-maxStoredComments:]
	}
	raw, err := MarshalList(next)
	if err != nil {
		return existing, nil, nil, err
	}
	flat := Flatten(next)
	var flatPtr *string
	if flat != "" {
		flatPtr = &flat
	}
	return next, raw, flatPtr, nil
}

// CommentsToJSON exports comments for API responses.
func CommentsToJSON(comments []Comment) []map[string]any {
	if len(comments) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		row := map[string]any{
			"id":          c.ID,
			"displayName": c.DisplayName,
			"body":        c.Body,
			"createdAt":   c.CreatedAt,
		}
		if c.UserID != nil && strings.TrimSpace(*c.UserID) != "" {
			row["userId"] = *c.UserID
		}
		if c.AvatarURL != nil && strings.TrimSpace(*c.AvatarURL) != "" {
			row["avatarUrl"] = *c.AvatarURL
		}
		if strings.TrimSpace(c.Source) != "" {
			row["source"] = c.Source
		}
		out = append(out, row)
	}
	return out
}