package caliper

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event types aligned with IMS Caliper 1.2 action vocabulary used by Lextures.
const (
	ActionLoggedIn       = "LoggedIn"
	ActionNavigatedTo    = "NavigatedTo"
	ActionCompleted      = "Completed"
	ActionSubmitted      = "Submitted"
	ActionGraded         = "Graded"
	ActionEnrolled       = "Enrolled"
)

// BuildInput constructs a Caliper Analytics 1.2 JSON-LD event.
type BuildInput struct {
	EventID     uuid.UUID
	EventType   string // e.g. SessionEvent, NavigationEvent
	Action      string
	ActorIRI    string
	ObjectIRI   string
	ObjectName  string
	CourseIRI   string
	Score       *float64
	Timestamp   time.Time
}

// Event is a minimal Caliper 1.2 event (JSON-LD).
type Event struct {
	Context     string         `json:"@context"`
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Action      string         `json:"action"`
	EventTime   string         `json:"eventTime"`
	Actor       EntityRef      `json:"actor"`
	Object      EntityRef      `json:"object"`
	Generated   *EntityRef     `json:"generated,omitempty"`
	Extensions  map[string]any `json:"extensions,omitempty"`
}

type EntityRef struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
}

// BuildEvent returns a Caliper event for the given input.
func BuildEvent(in BuildInput) Event {
	if in.EventID == uuid.Nil {
		in.EventID = uuid.New()
	}
	ts := in.Timestamp.UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	ev := Event{
		Context:   "http://purl.imsglobal.org/ctx/caliper/v1p2",
		ID:        "urn:uuid:" + in.EventID.String(),
		Type:      in.EventType,
		Action:    in.Action,
		EventTime: ts.Format(time.RFC3339),
		Actor: EntityRef{
			ID:   in.ActorIRI,
			Type: "Person",
		},
		Object: EntityRef{
			ID:   in.ObjectIRI,
			Type: "DigitalResource",
			Name: in.ObjectName,
		},
	}
	if in.Score != nil {
		ev.Extensions = map[string]any{"score": *in.Score}
	}
	if in.CourseIRI != "" {
		ev.Generated = &EntityRef{ID: in.CourseIRI, Type: "CourseOffering"}
	}
	return ev
}

// MarshalEvent JSON-encodes a Caliper event.
func MarshalEvent(ev Event) ([]byte, error) {
	return json.Marshal(ev)
}
