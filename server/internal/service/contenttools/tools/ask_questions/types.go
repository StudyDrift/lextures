package ask_questions

// Stance is the instructor-configured teaching stance.
type Stance string

const (
	StanceExplain  Stance = "explain"
	StanceSocratic Stance = "socratic"
	StanceHintOnly Stance = "hint_only"
)

// OffTopicPolicy controls off-topic handling.
type OffTopicPolicy string

const (
	OffTopicRedirect OffTopicPolicy = "redirect"
	OffTopicAnswer   OffTopicPolicy = "answer"
)

// Config is instructor-authored Ask Questions configuration.
type Config struct {
	Intro              string         `json:"intro,omitempty"`
	Placeholder        string         `json:"placeholder,omitempty"`
	Stance             Stance         `json:"stance"`
	GroundingNotes     string         `json:"groundingNotes,omitempty"`
	ExtraSourceURLs    []string       `json:"extraSourceUrls,omitempty"`
	OffTopicPolicy     OffTopicPolicy `json:"offTopicPolicy"`
	MaxQuestionsPerDay int            `json:"maxQuestionsPerDay"`
	MaxTurns           int            `json:"maxTurns"`
	ShowCitations      bool           `json:"showCitations"`
}

// Citation is a validated source handle stored on an assistant turn.
type Citation struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
}

// Turn is one conversation message.
type Turn struct {
	ID        string     `json:"id"`
	Role      string     `json:"role"` // user | assistant
	Text      string     `json:"text"`
	Citations []Citation `json:"citations,omitempty"`
	CreatedAt string     `json:"createdAt"`
	Tokens    *int       `json:"tokens,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// AskedToday tracks the per-day question cap.
type AskedToday struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// State is per-enrollment Ask Questions state.
type State struct {
	V          int         `json:"v"`
	Turns      []Turn      `json:"turns"`
	Summary    string      `json:"summary,omitempty"`
	AskedToday *AskedToday `json:"askedToday,omitempty"`
	Draft      string      `json:"draft,omitempty"`
}

// DefaultConfig returns config defaults from the manifest contract.
func DefaultConfig() Config {
	return Config{
		Stance:             StanceExplain,
		OffTopicPolicy:     OffTopicRedirect,
		MaxQuestionsPerDay: 20,
		MaxTurns:           40,
		ShowCitations:      true,
	}
}

// EmptyState returns a fresh conversation document.
func EmptyState() State {
	return State{V: 1, Turns: []Turn{}}
}
