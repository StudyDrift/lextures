// Package productfeedback defines enums and validation for in-app product feedback (plan FB0).
package productfeedback

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	MaxMessageLen = 5000
	PreviewLen    = 120
)

// Category classifies feedback intent.
type Category string

const (
	CategoryBug      Category = "bug"
	CategoryIdea     Category = "idea"
	CategoryQuestion Category = "question"
	CategoryPraise   Category = "praise"
	CategoryOther    Category = "other"
)

// Source is the client platform that submitted feedback.
type Source string

const (
	SourceWeb     Source = "web"
	SourceIOS     Source = "ios"
	SourceAndroid Source = "android"
)

// Status is the admin triage lifecycle state.
type Status string

const (
	StatusNew        Status = "new"
	StatusTriaged    Status = "triaged"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusWontFix    Status = "wont_fix"
	StatusArchived   Status = "archived"
)

// Context holds optional client metadata about where feedback was submitted.
type Context struct {
	Route     string `json:"route,omitempty"`
	Locale    string `json:"locale,omitempty"`
	Viewport  string `json:"viewport,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

// ParseCategory validates a category string (does not coerce to other).
func ParseCategory(raw string) (Category, error) {
	c := Category(strings.ToLower(strings.TrimSpace(raw)))
	switch c {
	case CategoryBug, CategoryIdea, CategoryQuestion, CategoryPraise, CategoryOther:
		return c, nil
	default:
		return "", fmt.Errorf("invalid category %q", raw)
	}
}

// NormalizeCategory coerces absent/invalid values to other (FR-5).
func NormalizeCategory(raw string) Category {
	c := Category(strings.ToLower(strings.TrimSpace(raw)))
	switch c {
	case CategoryBug, CategoryIdea, CategoryQuestion, CategoryPraise, CategoryOther:
		return c
	default:
		return CategoryOther
	}
}

// ParseSource validates a client-declared source.
func ParseSource(raw string) (Source, error) {
	s := Source(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case SourceWeb, SourceIOS, SourceAndroid:
		return s, nil
	default:
		return "", fmt.Errorf("invalid source %q", raw)
	}
}

// ParseStatus validates an admin status update.
func ParseStatus(raw string) (Status, error) {
	s := Status(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case StatusNew, StatusTriaged, StatusInProgress, StatusResolved, StatusWontFix, StatusArchived:
		return s, nil
	default:
		return "", fmt.Errorf("invalid status %q", raw)
	}
}

// IsTerminal returns true when moving to this status should set resolved_by/resolved_at.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusResolved, StatusWontFix, StatusArchived:
		return true
	default:
		return false
	}
}

// ValidateMessage trims, strips control characters, and enforces length (FR-2).
func ValidateMessage(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("message is required")
	}
	s = stripControlChars(s)
	if s == "" {
		return "", fmt.Errorf("message is required")
	}
	if len([]rune(s)) > MaxMessageLen {
		return "", fmt.Errorf("message exceeds %d characters", MaxMessageLen)
	}
	return s, nil
}

func stripControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ReconcileSource validates client source against the User-Agent when possible (FR-4).
func ReconcileSource(declared Source, userAgent string) Source {
	ua := strings.ToLower(userAgent)
	switch declared {
	case SourceIOS:
		if strings.Contains(ua, "android") {
			return SourceAndroid
		}
		return SourceIOS
	case SourceAndroid:
		if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
			return SourceIOS
		}
		return SourceAndroid
	default:
		if strings.Contains(ua, "lextures-ios") || strings.Contains(ua, "cfnetwork") && strings.Contains(ua, "darwin") {
			return SourceIOS
		}
		if strings.Contains(ua, "lextures-android") || strings.Contains(ua, "okhttp") {
			return SourceAndroid
		}
		return SourceWeb
	}
}

// MessagePreview returns a short excerpt for admin list views.
func MessagePreview(message string) string {
	runes := []rune(strings.TrimSpace(message))
	if len(runes) <= PreviewLen {
		return string(runes)
	}
	return string(runes[:PreviewLen]) + "…"
}
