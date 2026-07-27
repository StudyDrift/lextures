package media_checkpoints

import "github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"

// MediaSource identifies how media is referenced.
type MediaSource string

const (
	MediaSourceCourseFile MediaSource = "course_file"
	MediaSourceExternal   MediaSource = "external" // reserved; not supported in v1
)

// MediaKind is video or audio.
type MediaKind string

const (
	MediaKindVideo MediaKind = "video"
	MediaKindAudio MediaKind = "audio"
)

// TranscriptSource selects where the transcript comes from.
type TranscriptSource string

const (
	TranscriptFromCaptions TranscriptSource = "captions"
	TranscriptInline       TranscriptSource = "inline"
)

// QuestionType reuses the CT.11 set.
type QuestionType = inline_questions.QuestionType

const (
	TypeSingle    = inline_questions.TypeSingle
	TypeMulti     = inline_questions.TypeMulti
	TypeTrueFalse = inline_questions.TypeTrueFalse
	TypeShortText = inline_questions.TypeShortText
	TypeNumeric   = inline_questions.TypeNumeric
)

// ToleranceKind reuses CT.11 numeric tolerance kinds.
type ToleranceKind = inline_questions.ToleranceKind

const (
	ToleranceAbsolute = inline_questions.ToleranceAbsolute
	ToleranceRelative = inline_questions.ToleranceRelative
)

// Option is a choice option (correct/feedback are x-lex-sensitive).
type Option struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback,omitempty"`
}

// Tolerance is a numeric grading window.
type Tolerance struct {
	Kind  ToleranceKind `json:"kind"`
	Value float64       `json:"value"`
}

// Question is one checkpoint formative item.
type Question struct {
	Type            QuestionType `json:"type"`
	Prompt          string       `json:"prompt"`
	Options         []Option     `json:"options,omitempty"`
	AcceptedAnswers []string     `json:"acceptedAnswers,omitempty"`
	CorrectValue    *float64     `json:"correctValue,omitempty"`
	Tolerance       *Tolerance   `json:"tolerance,omitempty"`
}

// MediaRef points at a course file (or reserved external source).
type MediaRef struct {
	Source      MediaSource `json:"source"`
	FileID      string      `json:"fileId"`
	Kind        MediaKind   `json:"kind"`
	DurationSec float64     `json:"durationSec"`
	// URL is an optional resolved playback URL stored at authoring time
	// (same pattern as diagram_hotspot image.url). Required for learner playback
	// when the host cannot resolve course files from fileId alone.
	URL string `json:"url,omitempty"`
	// CaptionURL is an optional WebVTT track URL for captions.
	CaptionURL string `json:"captionUrl,omitempty"`
}

// Checkpoint is a timestamped question.
type Checkpoint struct {
	ID           string   `json:"id"`
	AtSec        float64  `json:"atSec"`
	Question     Question `json:"question"`
	Required     *bool    `json:"required,omitempty"` // default true
	Attempts     *int     `json:"attempts,omitempty"` // default 2
	ShowFeedback *bool    `json:"showFeedback,omitempty"`
}

// Config is instructor-authored Media Checkpoints configuration.
type Config struct {
	Media                      MediaRef         `json:"media"`
	CaptionsTrackID            string           `json:"captionsTrackId,omitempty"`
	TranscriptSource           TranscriptSource `json:"transcriptSource,omitempty"`
	TranscriptMarkdown         string           `json:"transcriptMarkdown,omitempty"`
	Checkpoints                []Checkpoint     `json:"checkpoints"`
	PreventSkipPastUnanswered  bool             `json:"preventSkipPastUnanswered"`
	PracticeOnly               bool             `json:"practiceOnly"`
	RequireCaptionsWhenPolicy  bool             `json:"requireCaptionsWhenPolicy,omitempty"`
}

// Attempt is one recorded submission for a checkpoint.
type Attempt struct {
	Value   any    `json:"value"`
	Correct bool   `json:"correct"`
	At      string `json:"at"`
}

// CheckpointAnswer is per-checkpoint learner progress.
type CheckpointAnswer struct {
	Attempts []Attempt `json:"attempts"`
	Done     bool      `json:"done"`
}

// State is per-enrollment Media Checkpoints state.
type State struct {
	V                 int                         `json:"v"`
	Answers           map[string]CheckpointAnswer `json:"answers"`
	WatchedSegments   [][2]float64                `json:"watchedSegments"`
	FurthestSec       float64                     `json:"furthestSec"`
	UsedTranscriptOnly bool                       `json:"usedTranscriptOnly,omitempty"`
	ScoreRaw          *float64                    `json:"scoreRaw,omitempty"`
	ScoreMax          *float64                    `json:"scoreMax,omitempty"`
	CompletedAt       string                      `json:"completedAt,omitempty"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		Media: MediaRef{
			Source: MediaSourceCourseFile,
			Kind:   MediaKindVideo,
		},
		TranscriptSource:          TranscriptInline,
		Checkpoints:               nil,
		PreventSkipPastUnanswered: false,
		PracticeOnly:              true,
	}
}

// EmptyState returns a fresh unanswered document.
func EmptyState() State {
	return State{
		V:               1,
		Answers:         map[string]CheckpointAnswer{},
		WatchedSegments: [][2]float64{},
		FurthestSec:     0,
	}
}

// CheckpointRequired reports whether the checkpoint must be answered to continue.
func CheckpointRequired(cp Checkpoint) bool {
	if cp.Required == nil {
		return true
	}
	return *cp.Required
}

// CheckpointAttempts returns the attempt limit (1–10, default 2).
func CheckpointAttempts(cp Checkpoint) int {
	if cp.Attempts == nil {
		return 2
	}
	n := *cp.Attempts
	if n < 1 {
		return 2
	}
	if n > 10 {
		return 10
	}
	return n
}

// CheckpointShowFeedback reports whether immediate feedback is shown (default true).
func CheckpointShowFeedback(cp Checkpoint) bool {
	if cp.ShowFeedback == nil {
		return true
	}
	return *cp.ShowFeedback
}
