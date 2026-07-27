package highlight_annotate

// PassageSource selects where the target passage comes from.
type PassageSource string

const (
	PassagePrecedingBlock PassageSource = "preceding_block"
	PassageInline         PassageSource = "inline"
	PassageSectionAnchor  PassageSource = "section_anchor"
)

// UnitGranularity is how the passage is segmented for keyboard navigation.
type UnitGranularity string

const (
	UnitSentence  UnitGranularity = "sentence"
	UnitParagraph UnitGranularity = "paragraph"
	UnitLine      UnitGranularity = "line"
)

// Tag is one annotation category.
type Tag struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// ExpectedRegion is an instructor-only expected annotation target (x-lex-sensitive).
type ExpectedRegion struct {
	TagID string `json:"tagId"`
	Quote string `json:"quote"`
}

// Config is instructor-authored Highlight & Annotate configuration.
type Config struct {
	Prompt           string          `json:"prompt"`
	PassageSource    PassageSource   `json:"passageSource"`
	PassageMarkdown  string          `json:"passageMarkdown,omitempty"`
	SectionAnchor    string          `json:"sectionAnchor,omitempty"`
	UnitGranularity  UnitGranularity `json:"unitGranularity"`
	Tags             []Tag           `json:"tags"`
	MinAnnotations   int             `json:"minAnnotations"`
	MaxAnnotations   int             `json:"maxAnnotations"`
	RequireNote      bool            `json:"requireNote"`
	ExpectedRegions  []ExpectedRegion `json:"expectedRegions,omitempty"`
}

// Anchor is a quote-plus-context selector (W3C TextQuoteSelector-inspired).
type Anchor struct {
	Prefix       string `json:"prefix"`
	Suffix       string `json:"suffix"`
	ApproxOffset int    `json:"approxOffset"`
	UnitIndex    *int   `json:"unitIndex,omitempty"`
}

// Annotation is one learner markup on the passage.
type Annotation struct {
	ID        string `json:"id"`
	TagID     string `json:"tagId"`
	Quote     string `json:"quote"`
	Anchor    Anchor `json:"anchor"`
	Note      string `json:"note,omitempty"`
	CreatedAt string `json:"createdAt"`
	Orphaned  bool   `json:"orphaned,omitempty"`
}

// State is per-enrollment Highlight & Annotate state.
type State struct {
	V           int          `json:"v"`
	Annotations []Annotation `json:"annotations"`
	CompletedAt string       `json:"completedAt,omitempty"`
}

// DefaultConfig returns manifest defaults.
func DefaultConfig() Config {
	return Config{
		PassageSource:   PassageInline,
		UnitGranularity: UnitSentence,
		MinAnnotations:  1,
		MaxAnnotations:  20,
		RequireNote:     false,
		Tags:            nil,
	}
}

// EmptyState returns a fresh document.
func EmptyState() State {
	return State{V: 1, Annotations: []Annotation{}}
}

// ActiveAnnotationCount returns non-empty annotations (all stored rows count toward completion).
func (s State) ActiveAnnotationCount() int {
	return len(s.Annotations)
}

// MeetsMinimum reports whether the learner has enough annotations for completion.
func (s State) MeetsMinimum(cfg Config) bool {
	min := cfg.MinAnnotations
	if min < 1 {
		min = 1
	}
	return s.ActiveAnnotationCount() >= min
}
