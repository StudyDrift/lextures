// Package webhooks defines outbound webhook event types and envelopes (plan 16.3).
package webhooks

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
)

const APIVersion = "2026-04-17"

// EventType is a registered outbound webhook event identifier.
type EventType string

const (
	EventGradePosted        EventType = "grade.posted"
	EventEnrollmentCreated  EventType = "enrollment.created"
	EventAssignmentSubmitted EventType = "assignment.submitted"
)

// AllEventTypes returns the supported event type strings sorted.
func AllEventTypes() []string {
	types := []EventType{
		EventGradePosted,
		EventEnrollmentCreated,
		EventAssignmentSubmitted,
	}
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, string(t))
	}
	sort.Strings(out)
	return out
}

// ValidEventTypes returns a set of known event type ids.
func ValidEventTypes() map[string]struct{} {
	out := make(map[string]struct{}, len(AllEventTypes()))
	for _, id := range AllEventTypes() {
		out[id] = struct{}{}
	}
	return out
}

// NormalizeEventTypes deduplicates and validates event types.
func NormalizeEventTypes(ids []string) ([]string, bool) {
	valid := ValidEventTypes()
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := raw
		if id == "" {
			continue
		}
		if _, ok := valid[id]; !ok {
			return nil, false
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, true
}

// Envelope is the canonical webhook payload shape.
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	APIVersion string          `json:"api_version"`
	CreatedAt  string          `json:"created_at"`
	Test       bool            `json:"test"`
	Data       json.RawMessage `json:"data"`
}

// NewEnvelope builds a delivery envelope with a fresh event id.
func NewEnvelope(eventType EventType, data any, test bool) (Envelope, []byte, error) {
	eventID := uuid.New()
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, nil, err
	}
	env := Envelope{
		EventID:    eventID.String(),
		EventType:  string(eventType),
		APIVersion: APIVersion,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Test:       test,
		Data:       raw,
	}
	body, err := json.Marshal(env)
	if err != nil {
		return Envelope{}, nil, err
	}
	return env, body, nil
}

// EventGroup describes a domain grouping for admin UI.
type EventGroup struct {
	Domain string   `json:"domain"`
	Types  []string `json:"types"`
}

// EventGroups returns event types grouped by domain prefix for admin UI.
func EventGroups() []EventGroup {
	return []EventGroup{
		{Domain: "Grades", Types: []string{string(EventGradePosted)}},
		{Domain: "Enrollments", Types: []string{string(EventEnrollmentCreated)}},
		{Domain: "Assignments", Types: []string{string(EventAssignmentSubmitted)}},
	}
}
