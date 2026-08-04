package coursechecklist

import "time"

// ChecklistResponse is the GET /checklist wire shape (CC.2 §9).
type ChecklistResponse struct {
	CourseCode        string              `json:"courseCode"`
	EngineVersion     int                 `json:"engineVersion"`
	CatalogVersion    string              `json:"catalogVersion"`
	ComputedAt        time.Time           `json:"computedAt"`
	Stale             bool                `json:"stale"`
	EvidenceTruncated bool                `json:"evidenceTruncated"`
	Summary           ChecklistSummary    `json:"summary"`
	Categories        []ChecklistCategory `json:"categories"`
	Dismissed         []ChecklistItem     `json:"dismissed"`
}

// ChecklistCategory groups items for one registry category.
type ChecklistCategory struct {
	ID       string          `json:"id"`
	TitleKey string          `json:"titleKey"`
	Title    string          `json:"title"`
	Items    []ChecklistItem `json:"items"`
}

// ChecklistItem is one checklist finding plus optional dismissal metadata.
type ChecklistItem struct {
	ID         string              `json:"id"`
	TitleKey   string              `json:"titleKey"`
	Title      string              `json:"title"`
	WhyKey     string              `json:"whyKey"`
	Why        string              `json:"why"`
	Tier       Tier                `json:"tier"`
	Status     Status              `json:"status"`
	Detail     *string             `json:"detail"`
	Progress   *Progress           `json:"progress"`
	Sources    []string            `json:"sources"`
	HelpRef    *string             `json:"helpRef"`
	Target     *ChecklistNavTarget `json:"target"`
	Evidence   *ChecklistEvidence  `json:"evidence"`
	Dismissal  *ChecklistDismissal `json:"dismissal"`
}

// ChecklistNavTarget is the client navigation target for an item.
type ChecklistNavTarget struct {
	Route  string  `json:"route"`
	Anchor *string `json:"anchor"`
}

// ChecklistEvidence is the expandable evidence table.
type ChecklistEvidence struct {
	Columns     []string              `json:"columns"`
	Rows        []ChecklistEvidenceRow `json:"rows"`
	TruncatedAt *int                  `json:"truncatedAt"`
}

// ChecklistEvidenceRow is one evidence table row.
type ChecklistEvidenceRow struct {
	Label    string              `json:"label"`
	Sublabel *string             `json:"sublabel"`
	Status   string              `json:"status"`
	Target   *ChecklistNavTarget `json:"target"`
}

// ChecklistDismissal is course-scoped dismissal metadata.
type ChecklistDismissal struct {
	DismissedAt   time.Time `json:"dismissedAt"`
	ByUserID      string    `json:"byUserId"`
	ByDisplayName string    `json:"byDisplayName"`
	Reason        string    `json:"reason"`
	Note          string    `json:"note"`
}

// ChecklistSummary is the badge / progress counters (FR-14).
type ChecklistSummary struct {
	OutstandingEssential int       `json:"outstandingEssential"`
	OutstandingTotal     int       `json:"outstandingTotal"`
	Done                 int       `json:"done"`
	Total                int       `json:"total"`
	Dismissed            int       `json:"dismissed"`
	ComputedAt           time.Time `json:"computedAt"`
	Stale                bool      `json:"stale"`
}

// HistoryEvent is one checklist audit event for GET /history.
type HistoryEvent struct {
	ID          string    `json:"id"`
	ItemID      string    `json:"itemId"`
	Action      string    `json:"action"`
	ActorUserID *string   `json:"actorUserId"`
	Reason      string    `json:"reason"`
	OccurredAt  time.Time `json:"occurredAt"`
}

// HistoryResponse is GET /checklist/history.
type HistoryResponse struct {
	CourseCode     string         `json:"courseCode"`
	EngineVersion  int            `json:"engineVersion"`
	CatalogVersion string         `json:"catalogVersion"`
	Events         []HistoryEvent `json:"events"`
}

// DismissRequest is POST .../dismiss body.
type DismissRequest struct {
	Reason string `json:"reason"`
	Note   string `json:"note"`
}

// snapshotPayload is the JSON stored in course_checklist_snapshots.payload.
type snapshotPayload struct {
	Result            Result `json:"result"`
	EvidenceTruncated bool   `json:"evidenceTruncated"`
}
