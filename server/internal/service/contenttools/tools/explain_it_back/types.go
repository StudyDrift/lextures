package explain_it_back

// FeedbackStyle controls the tone of AI formative feedback.
type FeedbackStyle string

const (
	FeedbackEncouraging FeedbackStyle = "encouraging"
	FeedbackNeutral     FeedbackStyle = "neutral"
	FeedbackSocratic    FeedbackStyle = "socratic"
)

// FeedbackMode is how feedback was produced.
type FeedbackMode string

const (
	FeedbackModeAI     FeedbackMode = "ai"
	FeedbackModeReview FeedbackMode = "review"
)

// KeyPoint is one author-defined idea the explanation should address.
type KeyPoint struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Config is instructor-authored Explain It Back configuration.
type Config struct {
	Prompt                     string        `json:"prompt"`
	MinWords                   int           `json:"minWords"`
	MaxWords                   int           `json:"maxWords"`
	KeyPoints                  []KeyPoint    `json:"keyPoints"`
	RevealKeyPointsAfterSubmit bool          `json:"revealKeyPointsAfterSubmit"`
	AIFeedback                 bool          `json:"aiFeedback"`
	FeedbackStyle              FeedbackStyle `json:"feedbackStyle"`
	Attempts                   int           `json:"attempts"`
	IncludeProbeQuestion       bool          `json:"includeProbeQuestion"`
	AllowInstructorNote        bool          `json:"allowInstructorNote"`
	MaxSubmissionsPerDay       int           `json:"maxSubmissionsPerDay"`
}

// Feedback is formative feedback for one attempt.
type Feedback struct {
	Covered    []string     `json:"covered"`
	Missing    []string     `json:"missing"`
	Strength   string       `json:"strength"`
	Suggestion string       `json:"suggestion"`
	Probe      string       `json:"probe,omitempty"`
	Mode       FeedbackMode `json:"mode"`
}

// Attempt is one learner submission.
type Attempt struct {
	At       string    `json:"at"`
	Text     string    `json:"text"`
	Feedback *Feedback `json:"feedback,omitempty"`
}

// InstructorNote is a short note left for one learner.
type InstructorNote struct {
	Text string `json:"text"`
	At   string `json:"at"`
	By   string `json:"by"`
}

// SubmittedToday tracks the per-day submission cap.
type SubmittedToday struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// State is per-enrollment Explain It Back state.
type State struct {
	V               int              `json:"v"`
	Draft           string           `json:"draft,omitempty"`
	Attempts        []Attempt        `json:"attempts"`
	InstructorNote  *InstructorNote  `json:"instructorNote,omitempty"`
	SubmittedToday  *SubmittedToday  `json:"submittedToday,omitempty"`
	CompletedAt     string           `json:"completedAt,omitempty"`
	CrisisAcknowledged bool          `json:"crisisAcknowledged,omitempty"`
}

// DefaultConfig returns config defaults from the manifest contract.
func DefaultConfig() Config {
	return Config{
		MinWords:                   25,
		MaxWords:                   150,
		KeyPoints:                  []KeyPoint{},
		RevealKeyPointsAfterSubmit: true,
		AIFeedback:                 true,
		FeedbackStyle:              FeedbackEncouraging,
		Attempts:                   3,
		IncludeProbeQuestion:       true,
		AllowInstructorNote:        true,
		MaxSubmissionsPerDay:       10,
	}
}

// EmptyState returns a fresh learner document.
func EmptyState() State {
	return State{V: 1, Attempts: []Attempt{}}
}

// ReviewAcknowledgement is the learner-facing message when AI feedback is unavailable.
const ReviewAcknowledgement = "Thanks — your explanation was saved for your instructor to review. This is practice, not a grade."

// CrisisSupportMessage is shown instead of normal feedback when a crisis signal is detected.
const CrisisSupportMessage = "If you are struggling, please reach out to someone you trust or your school's support resources. Your writing was not sent for AI feedback."
