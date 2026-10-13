package parentprogress

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/models/gradecomment"
	repo "github.com/lextures/lextures/server/internal/repos/parentprogress"
)

// Section is one learner block in the weekly email.
type Section struct {
	Name         string
	Completed    []string
	DueNext      []string
	GradeChanges []string
	TimeSpent    string
	TimeSeconds  int
	Comments     []string
}

// HasActivity is true when the learner finished work, changed a grade, spent time, or received a comment.
func (s Section) HasActivity() bool {
	return len(s.Completed) > 0 || len(s.GradeChanges) > 0 || s.TimeSeconds > 0 || len(s.Comments) > 0
}

// AnyActivity reports whether any learner section has something to report.
func AnyActivity(sections []Section) bool {
	for _, s := range sections {
		if s.HasActivity() {
			return true
		}
	}
	return false
}

// BuildSection turns repository rows into one learner section.
func BuildSection(name string, completed, due []repo.Item, grades []repo.GradeChange, comments []repo.Comment, timeSeconds int, windowStart time.Time) Section {
	s := Section{Name: name, TimeSeconds: timeSeconds, TimeSpent: formatDuration(timeSeconds)}
	seen := map[string]struct{}{}
	for _, it := range completed {
		line := it.Course + " — " + it.Title
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		s.Completed = append(s.Completed, line)
	}
	for _, it := range due {
		s.DueNext = append(s.DueNext, fmt.Sprintf("%s — %s (due %s)", it.Course, it.Title, it.When.Format("Mon Jan 2")))
	}
	for _, g := range grades {
		if strings.TrimSpace(g.Previous) != "" && g.Previous != g.Next {
			s.GradeChanges = append(s.GradeChanges, fmt.Sprintf("%s — %s: %s → %s", g.Course, g.Title, g.Previous, g.Next))
			continue
		}
		s.GradeChanges = append(s.GradeChanges, fmt.Sprintf("%s — %s: %s", g.Course, g.Title, g.Next))
	}
	for _, raw := range comments {
		parsed, err := gradecomment.ParseList([]byte(raw.Body))
		if err != nil {
			continue
		}
		for i := len(parsed) - 1; i >= 0; i-- {
			body := strings.TrimSpace(parsed[i].Body)
			if body == "" {
				continue
			}
			if parsed[i].CreatedAt != "" {
				at, perr := time.Parse(time.RFC3339, parsed[i].CreatedAt)
				if perr == nil && at.Before(windowStart) {
					continue
				}
			}
			s.Comments = append(s.Comments, fmt.Sprintf("%s — %s: %s", raw.Course, raw.Title, oneLine(body)))
			break
		}
	}
	return s
}

// Message is the rendered weekly email.
type Message struct {
	Subject  string
	BodyText string
	BodyHTML string
	NoteOnly bool
}

// Compose builds the email. When noteOnly is set, the body is the short empty-week note.
func Compose(parentName string, sections []Section, noteOnly bool) Message {
	if strings.TrimSpace(parentName) == "" {
		parentName = "there"
	}
	if noteOnly {
		text := fmt.Sprintf("Hi %s,\n\nNo learning activity this week.\n", parentName)
		html := fmt.Sprintf("<p>Hi %s,</p><p>No learning activity this week.</p>", template.HTMLEscapeString(parentName))
		return Message{Subject: "No learning activity this week", BodyText: text, BodyHTML: html, NoteOnly: true}
	}
	var text strings.Builder
	var html strings.Builder
	fmt.Fprintf(&text, "Hi %s,\n\nHere is this week's progress.\n", parentName)
	fmt.Fprintf(&html, "<p>Hi %s,</p><p>Here is this week&#39;s progress.</p>", template.HTMLEscapeString(parentName))
	for _, s := range sections {
		fmt.Fprintf(&text, "\n%s\n", s.Name)
		fmt.Fprintf(&html, "<h2 style=\"font-size:16px;margin:20px 0 8px;\">%s</h2>", template.HTMLEscapeString(s.Name))
		writeBlock(&text, &html, "Completed", s.Completed)
		writeBlock(&text, &html, "Due next week", s.DueNext)
		writeBlock(&text, &html, "Grade changes", s.GradeChanges)
		writeLine(&text, &html, "Time spent", s.TimeSpent)
		writeBlock(&text, &html, "New comments", s.Comments)
	}
	return Message{Subject: "Weekly progress", BodyText: text.String(), BodyHTML: html.String()}
}

func writeBlock(text, html *strings.Builder, label string, lines []string) {
	if len(lines) == 0 {
		writeLine(text, html, label, "None")
		return
	}
	fmt.Fprintf(text, "%s:\n", label)
	fmt.Fprintf(html, "<p style=\"margin:8px 0 4px;\"><strong>%s</strong></p><ul style=\"margin:0 0 8px 18px;padding:0;\">", template.HTMLEscapeString(label))
	for _, line := range lines {
		fmt.Fprintf(text, "• %s\n", line)
		fmt.Fprintf(html, "<li>%s</li>", template.HTMLEscapeString(line))
	}
	html.WriteString("</ul>")
}

func writeLine(text, html *strings.Builder, label, value string) {
	fmt.Fprintf(text, "%s: %s\n", label, value)
	fmt.Fprintf(html, "<p style=\"margin:4px 0;\"><strong>%s:</strong> %s</p>", template.HTMLEscapeString(label), template.HTMLEscapeString(value))
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "No time recorded"
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if hours > 0 && mins > 0 {
		return fmt.Sprintf("%d h %d min", hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%d h", hours)
	}
	if mins == 0 {
		mins = 1
	}
	return fmt.Sprintf("%d min", mins)
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 180 {
		return s[:177] + "..."
	}
	return s
}
