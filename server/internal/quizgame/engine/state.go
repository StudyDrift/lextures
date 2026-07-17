package engine

import "time"

// SnapshotQuestion is one frozen question in kit_snapshot (server-side; may include correctness).
type SnapshotQuestion struct {
	ID               string         `json:"id"`
	Position         int            `json:"position"`
	QuestionType     string         `json:"questionType"`
	Prompt           string         `json:"prompt"`
	PromptMediaRef   *string        `json:"promptMediaRef,omitempty"`
	PromptMediaAlt   *string        `json:"promptMediaAlt,omitempty"`
	Options          []Option       `json:"options"`
	CorrectAnswer    map[string]any `json:"correctAnswer,omitempty"`
	TimeLimitSeconds int            `json:"timeLimitSeconds"`
	PointsStyle      string         `json:"pointsStyle"`
	AnswerShuffle    bool           `json:"answerShuffle"`
	Explanation      *string        `json:"explanation,omitempty"`
	SourceQuestionID *string        `json:"sourceQuestionId,omitempty"`
}

// Option is a choice without relying on repo types (engine is pure).
type Option struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	MediaRef  *string `json:"mediaRef,omitempty"`
	MediaAlt  *string `json:"mediaAlt,omitempty"`
	IsCorrect bool    `json:"isCorrect"`
}

// KitSnapshot is the frozen kit payload stored on the session.
type KitSnapshot struct {
	KitID     string             `json:"kitId"`
	Title     string             `json:"title"`
	Questions []SnapshotQuestion `json:"questions"`
}

// State is the in-memory / reducer view of a game (reconstructable from DB).
type State struct {
	SessionID     string
	Status        Status
	Phase         Phase
	Pacing        Pacing
	QuestionIndex int // -1 = lobby
	OpenedAt      *time.Time
	Deadline      *time.Time
	QuestionCount int
	HostPaused    bool
	ResumePhase   Phase // phase to restore after host reconnect
}

// Event is an append-only log entry produced by a transition.
type Event struct {
	Type    string
	Payload map[string]any
}

// PublicQuestion is the player/projector-safe question (no correctness).
type PublicQuestion struct {
	Index            int      `json:"index"`
	QuestionType     string   `json:"questionType"`
	Prompt           string   `json:"prompt"`
	PromptMediaRef   *string  `json:"promptMediaRef,omitempty"`
	PromptMediaAlt   *string  `json:"promptMediaAlt,omitempty"`
	Options          []Option `json:"options"`
	TimeLimitSeconds int      `json:"timeLimitSeconds"`
	PointsStyle      string   `json:"pointsStyle"`
}

// ToPublicQuestion strips correctness and optionally shuffles option order (caller shuffles).
func ToPublicQuestion(q SnapshotQuestion, index int) PublicQuestion {
	opts := make([]Option, len(q.Options))
	for i, o := range q.Options {
		opts[i] = Option{
			ID:       o.ID,
			Text:     o.Text,
			MediaRef: o.MediaRef,
			MediaAlt: o.MediaAlt,
			// IsCorrect intentionally omitted / false for wire
		}
	}
	return PublicQuestion{
		Index:            index,
		QuestionType:     q.QuestionType,
		Prompt:           q.Prompt,
		PromptMediaRef:   q.PromptMediaRef,
		PromptMediaAlt:   q.PromptMediaAlt,
		Options:          opts,
		TimeLimitSeconds: q.TimeLimitSeconds,
		PointsStyle:      q.PointsStyle,
	}
}
