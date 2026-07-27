package context

import (
	"time"

	"github.com/google/uuid"
)

// ExtractionVersion bumps when HTML/PDF parsers change so caches invalidate.
const ExtractionVersion = 1

// Default token / fetch budgets (FR-14, FR-4).
const (
	DefaultRequestContextTokens  = 8000
	DefaultRequestCompletionToks = 1000
	DefaultDailyCallsPerUser     = 50
	DefaultTopKLinks             = 3
	DefaultCacheTTL              = 7 * 24 * time.Hour
	MaxFetchBytes                = 5 * 1024 * 1024
	MaxRedirects                 = 3
	FetchTimeout                 = 10 * time.Second
)

// UserAgent identifies Lextures fetches (FR-5).
const UserAgent = "LexturesContentToolsBot/1.0 (+https://lextures.com/bot)"

// SegmentKind is a typed pack segment.
type SegmentKind string

const (
	KindSection  SegmentKind = "section"
	KindActivity SegmentKind = "activity"
	KindCourse   SegmentKind = "course"
	KindFile     SegmentKind = "file"
	KindLink     SegmentKind = "link"
	KindNote     SegmentKind = "note"
)

// CitationKind is a renderable citation handle kind (FR-11).
type CitationKind string

const (
	CiteSection CitationKind = "section"
	CiteFile    CitationKind = "file"
	CiteLink    CitationKind = "link"
)

// ContextSegment is one ordered, attributed pack segment (FR-1).
type ContextSegment struct {
	Kind   SegmentKind `json:"kind"`
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	URL    string      `json:"url,omitempty"`
	Lang   string      `json:"lang,omitempty"`
	Text   string      `json:"text"`
	Tokens int         `json:"tokens"`
}

// PendingSource is a discovered source not yet ready for grounding.
type PendingSource struct {
	URL    string `json:"url"`
	Status string `json:"status"` // pending | blocked | failed
	Reason string `json:"reason,omitempty"`
}

// ContextPack is the assembled grounded context for a tool instance (FR-1).
type ContextPack struct {
	InstanceID     uuid.UUID        `json:"instanceId"`
	Segments       []ContextSegment `json:"segments"`
	PendingSources []PendingSource  `json:"pendingSources"`
	TotalTokens    int              `json:"totalTokens"`
	VariantID      *uuid.UUID       `json:"variantId,omitempty"`
}

// Citation is a stable footnote handle (FR-11).
type Citation struct {
	Kind  CitationKind `json:"kind"`
	ID    string       `json:"id"`
	Title string       `json:"title"`
	URL   string       `json:"url,omitempty"`
	Loc   string       `json:"loc,omitempty"`
}

// BuildOpts controls pack assembly.
type BuildOpts struct {
	TokenBudget      int
	LearnerUserID    *uuid.UUID
	Query            string // optional: for ranking / orchestrated retrieval
	TopKLinks        int
	EnqueueIngest    bool
	ForceSyncIngest  bool // tests / instructor re-ingest
	ServeVariantText string
	ServeVariantID   *uuid.UUID
	PinnedNotes      string
	ConfigLinks      []string
	SkipLinkIngest   bool
}

// SourceStatus values persisted on link sources.
const (
	StatusPending     = "pending"
	StatusReady       = "ready"
	StatusBlocked     = "blocked"
	StatusFailed      = "failed"
	StatusUnsupported = "unsupported"
)

// Origin values for activity sources.
const (
	OriginBodyLink   = "body_link"
	OriginConfigLink = "config_link"
	OriginCourseFile = "course_file"
)

// LinkIngestion modes (FR-16).
const (
	IngestOff       = "off"
	IngestAllowlist = "allowlist"
	IngestPublic    = "public"
)

// ActivitySourceView is instructor-facing corpus row (FR-17).
type ActivitySourceView struct {
	ID               uuid.UUID  `json:"id"`
	SourceID         *uuid.UUID `json:"sourceId,omitempty"`
	URL              string     `json:"url"`
	Title            string     `json:"title,omitempty"`
	Host             string     `json:"host,omitempty"`
	Origin           string     `json:"origin"`
	Status           string     `json:"status"`
	Error            string     `json:"error,omitempty"`
	FetchedAt        *time.Time `json:"fetchedAt,omitempty"`
	ByteSize         *int       `json:"byteSize,omitempty"`
	Excluded         bool       `json:"excluded"`
	ExtractedText    string     `json:"extractedText,omitempty"`
	ExtractionQuality string    `json:"extractionQuality,omitempty"`
}
