package credentials

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildAchievementSubject constructs an Open Badges 3.0 credentialSubject map.
func BuildAchievementSubject(recipientID uuid.UUID, learnerName, achievementName, description, criteriaNarrative string) map[string]any {
	achievement := map[string]any{
		"type":        []string{"Achievement"},
		"name":        achievementName,
		"description": description,
		"criteria": map[string]any{
			"narrative": criteriaNarrative,
		},
	}
	return map[string]any{
		"type":  []string{"AchievementSubject"},
		"id":    fmt.Sprintf("urn:uuid:user:%s", recipientID.String()),
		"name":  strings.TrimSpace(learnerName),
		"achievement": achievement,
	}
}

// CriteriaNarrativeForSource returns human-readable completion criteria.
func CriteriaNarrativeForSource(sourceType string) string {
	switch sourceType {
	case "path":
		return "Completed every course in the learning path."
	case "ceu":
		return "Met the required contact hours for continuing education credit."
	default:
		return "Completed all required items in the course."
	}
}

// DefaultAchievementName builds a display title for a credential.
func DefaultAchievementName(sourceType, title string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = "Course Completion"
	}
	switch sourceType {
	case "path":
		if !strings.Contains(strings.ToLower(name), "learning path") {
			return name + " — Learning Path"
		}
	}
	return name
}

// IssuanceTimestamp returns the RFC3339 issuance time.
func IssuanceTimestamp(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}