package flashcards

// Rating is a self-rated recall grade (maps to SM-2 quality via service/srs).
type Rating string

const (
	RatingAgain Rating = "again"
	RatingHard  Rating = "hard"
	RatingGood  Rating = "good"
	RatingEasy  Rating = "easy"
)

// Card is one author-defined flashcard with a stable id.
type Card struct {
	ID        string `json:"id"`
	Front     string `json:"front"`
	Back      string `json:"back"`
	FrontLang string `json:"frontLang,omitempty"`
	BackLang  string `json:"backLang,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
	ImageAlt  string `json:"imageAlt,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

// Config is instructor-authored Flashcards configuration.
type Config struct {
	Title            string `json:"title,omitempty"`
	Cards            []Card `json:"cards"`
	ReversePractice  bool   `json:"reversePractice"`
	SessionCap       int    `json:"sessionCap"`
	Shuffle          bool   `json:"shuffle"`
	RequireFirstPass bool   `json:"requireFirstPass"`
}

// CardProgress is per-card progress stored in tool state (scheduling truth is SRS).
type CardProgress struct {
	Seen        int     `json:"seen"`
	LastRating  *Rating `json:"lastRating,omitempty"`
	LastSeenAt  string  `json:"lastSeenAt,omitempty"`
	FirstRating *Rating `json:"firstRating,omitempty"`
}

// SessionRecord is one completed (or in-progress) session summary.
type SessionRecord struct {
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt,omitempty"`
	Reviewed  int    `json:"reviewed"`
}

// QueueItem is one card/side pair in the active session queue.
type QueueItem struct {
	CardID string `json:"cardId"`
	Side   string `json:"side"` // forward | reverse
}

// ActiveSession is the in-progress session bookkeeping.
type ActiveSession struct {
	StartedAt string      `json:"startedAt"`
	Queue     []QueueItem `json:"queue"`
	Index     int         `json:"index"`
	Reviewed  int         `json:"reviewed"`
	Revealed  bool        `json:"revealed"`
}

// State is per-enrollment flashcards progress.
type State struct {
	V                    int                      `json:"v"`
	Cards                map[string]CardProgress  `json:"cards,omitempty"`
	Sessions             []SessionRecord          `json:"sessions,omitempty"`
	// ActiveSession must not use omitempty: session end sets it to nil, and
	// MergeStateJSON only clears keys present in the patch. Omitting the field
	// left a stale active session after the final rate (E2E Start button gone).
	ActiveSession        *ActiveSession           `json:"activeSession"`
	FirstPassCompletedAt string                   `json:"firstPassCompletedAt,omitempty"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		ReversePractice:  false,
		SessionCap:       20,
		Shuffle:          true,
		RequireFirstPass: true,
	}
}

// EmptyState returns a fresh progress document.
func EmptyState() State {
	return State{V: 1, Cards: map[string]CardProgress{}}
}

// ValidRating reports whether r is a known grade.
func ValidRating(r Rating) bool {
	switch r {
	case RatingAgain, RatingHard, RatingGood, RatingEasy:
		return true
	default:
		return false
	}
}

// ValidSide reports whether side is forward or reverse.
func ValidSide(side string) bool {
	return side == SideForward || side == SideReverse
}
