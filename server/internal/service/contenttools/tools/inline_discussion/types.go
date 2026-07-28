package inline_discussion

// AnonymityMode controls peer visibility of author identity.
type AnonymityMode string

const (
	AnonymityNamed            AnonymityMode = "named"
	AnonymityAnonymousToPeers AnonymityMode = "anonymous_to_peers"
)

// SortOrder controls thread listing order.
type SortOrder string

const (
	SortOldest SortOrder = "oldest"
	SortNewest SortOrder = "newest"
)

// Config is instructor-authored Inline Discussion configuration.
type Config struct {
	Prompt            string        `json:"prompt"`
	PostBeforeYouSee  bool          `json:"postBeforeYouSee"`
	AllowReplies      bool          `json:"allowReplies"`
	RequiredPosts     int           `json:"requiredPosts"`
	RequiredReplies   int           `json:"requiredReplies"`
	Anonymity         AnonymityMode `json:"anonymity"`
	EditWindowMinutes int           `json:"editWindowMinutes"`
	AllowDelete       bool          `json:"allowDelete"`
	Sort              SortOrder     `json:"sort"`
	PageSize          int           `json:"pageSize"`
}

// State is per-enrollment participation only; posts live in discussion tables.
type State struct {
	V           int      `json:"v"`
	ThreadID    string   `json:"threadId,omitempty"`
	MyPostIDs   []string `json:"myPostIds,omitempty"`
	MyReplyIDs  []string `json:"myReplyIds,omitempty"`
	LastReadAt  string   `json:"lastReadAt,omitempty"`
	CompletedAt string   `json:"completedAt,omitempty"`
	Draft       string   `json:"draft,omitempty"`
}

// PostMeta is stored in TipTap doc attrs.lex (no migration).
type PostMeta struct {
	Removed    bool   `json:"removed,omitempty"`
	Endorsed   bool   `json:"endorsed,omitempty"`
	EndorsedAt string `json:"endorsedAt,omitempty"`
	EndorsedBy string `json:"endorsedBy,omitempty"`
	EditedAt   string `json:"editedAt,omitempty"`
}

// DefaultConfig returns manifest defaults (HE-oriented named anonymity).
func DefaultConfig() Config {
	return Config{
		PostBeforeYouSee:  true,
		AllowReplies:      true,
		RequiredPosts:     1,
		RequiredReplies:   0,
		Anonymity:         AnonymityNamed,
		EditWindowMinutes: 5,
		AllowDelete:       true,
		Sort:              SortOldest,
		PageSize:          20,
	}
}

// EmptyState returns a fresh participation document.
func EmptyState() State {
	return State{V: 1, MyPostIDs: []string{}, MyReplyIDs: []string{}}
}
