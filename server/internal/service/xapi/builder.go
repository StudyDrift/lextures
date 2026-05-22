package xapi

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	VerbLaunched    = "http://adlnet.gov/expapi/verbs/launched"
	VerbExperienced = "http://adlnet.gov/expapi/verbs/experienced"
	VerbAnswered    = "http://adlnet.gov/expapi/verbs/answered"
	VerbPassed      = "http://adlnet.gov/expapi/verbs/passed"
	VerbFailed      = "http://adlnet.gov/expapi/verbs/failed"
	VerbCompleted   = "http://adlnet.gov/expapi/verbs/completed"
	VerbSubmitted   = "http://adlnet.gov/expapi/verbs/submitted"
	VerbReceived    = "http://adlnet.gov/expapi/verbs/received"
)

// BuildInput is the data needed to construct a standards-shaped xAPI 1.0.3 statement.
type BuildInput struct {
	StatementID uuid.UUID
	ActorEmail  string
	ActorName   string
	Anonymize   bool
	VerbID      string
	ObjectID    string
	ObjectType  string
	ObjectTitle string
	CourseIRI   string
	Score       *float64
	Success     *bool
	Timestamp   time.Time
}

// Statement is a minimal xAPI 1.0.3 statement document.
type Statement struct {
	ID       string         `json:"id"`
	Actor    Actor          `json:"actor"`
	Verb     Verb           `json:"verb"`
	Object   ActivityObject `json:"object"`
	Result   *Result        `json:"result,omitempty"`
	Context  *Context       `json:"context,omitempty"`
	Stored   string         `json:"stored,omitempty"`
}

type Actor struct {
	Mbox string  `json:"mbox,omitempty"`
	Name *string `json:"name,omitempty"`
}

type Verb struct {
	ID      string            `json:"id"`
	Display map[string]string `json:"display,omitempty"`
}

type ActivityObject struct {
	ID         string                 `json:"id"`
	ObjectType string                 `json:"objectType,omitempty"`
	Definition *ActivityDefinition    `json:"definition,omitempty"`
}

type ActivityDefinition struct {
	Name        map[string]string `json:"name,omitempty"`
	Type        string            `json:"type,omitempty"`
}

type Result struct {
	Score   *Score  `json:"score,omitempty"`
	Success *bool   `json:"success,omitempty"`
}

type Score struct {
	Scaled *float64 `json:"scaled,omitempty"`
	Raw    *float64 `json:"raw,omitempty"`
}

type Context struct {
	ContextActivities *ContextActivities `json:"contextActivities,omitempty"`
}

type ContextActivities struct {
	Parent []ActivityRef `json:"parent,omitempty"`
}

type ActivityRef struct {
	ID string `json:"id"`
}

// BuildStatement constructs an xAPI statement from BuildInput.
func BuildStatement(in BuildInput) Statement {
	if in.StatementID == uuid.Nil {
		in.StatementID = uuid.New()
	}
	ts := in.Timestamp.UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	name := strings.TrimSpace(in.ActorName)
	var namePtr *string
	if name != "" && !in.Anonymize {
		namePtr = &name
	}
	stmt := Statement{
		ID: in.StatementID.String(),
		Actor: Actor{
			Mbox: ActorMbox(in.ActorEmail, in.Anonymize),
			Name: namePtr,
		},
		Verb: Verb{ID: in.VerbID},
		Object: ActivityObject{
			ID:         in.ObjectID,
			ObjectType: in.ObjectType,
		},
		Stored: ts.Format(time.RFC3339),
	}
	if in.ObjectTitle != "" {
		stmt.Object.Definition = &ActivityDefinition{
			Name: map[string]string{"en-US": in.ObjectTitle},
		}
	}
	if in.Score != nil || in.Success != nil {
		res := &Result{Success: in.Success}
		if in.Score != nil {
			res.Score = &Score{Scaled: in.Score}
		}
		stmt.Result = res
	}
	if in.CourseIRI != "" {
		stmt.Context = &Context{
			ContextActivities: &ContextActivities{
				Parent: []ActivityRef{{ID: in.CourseIRI}},
			},
		}
	}
	return stmt
}

// MarshalStatement JSON-encodes a statement.
func MarshalStatement(stmt Statement) ([]byte, error) {
	return json.Marshal(stmt)
}
