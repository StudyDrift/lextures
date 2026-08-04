package coursechecklist

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CourseSnapshot is the in-memory course view evaluators receive (FR-6).
// Fields are populated according to the DataNeeds passed to LoadSnapshot.
type CourseSnapshot struct {
	CourseCode string
	CourseID   uuid.UUID

	// Course (DataNeedCourse)
	Title                   string
	Description             string
	Published               bool
	StartsAt                *time.Time
	EndsAt                  *time.Time
	VisibleFrom             *time.Time
	HiddenAt                *time.Time
	CourseTimezone          *string
	ScheduleMode            string
	SectionsEnabled         bool
	FeedEnabled             bool
	FilesEnabled            bool
	SbgEnabled              bool
	SbgProficiencyScaleJSON json.RawMessage
	ModuleGatingEnabled     bool
	StandardsEnabled        bool
	CourseType              string
	CourseMode              string
	HeroImageURL            *string
	CourseHomeLanding       string
	CourseHomeContentItemID *string
	CreatedAt               time.Time
	FeaturesReviewedAt      *time.Time
	GradingSchemeID         *uuid.UUID
	CatalogLanguage         string
	HomeschoolMode          bool // true when OrgID is nil (personal / single-instructor)
	ParentPortalEnabled     bool
	OrgIsK12                bool
	CreatorUserID           *uuid.UUID
	Features                CourseFeatures

	// Structure (DataNeedStructure)
	StructureItems []StructureItem

	// Item detail metadata (DataNeedItemMeta)
	ItemMeta map[uuid.UUID]ItemMeta

	// Syllabus (DataNeedSyllabus)
	SyllabusSections          []SyllabusSectionSnap
	SyllabusMalformed         bool
	AcceptanceDecidedAt       *time.Time
	RequireSyllabusAcceptance bool
	SyllabusCheckedTruncated  bool

	// Outcomes (DataNeedOutcomes)
	Outcomes     []OutcomeSnap
	OutcomeLinks []OutcomeLinkSnap

	// Shared gradable set computed once after structure (+ optional item_meta) load (CC.4 FR-26).
	GradableItems []GradableItem
	// GradableComputedAt is non-zero when GradableItems was populated by LoadSnapshot.
	GradableComputed bool

	// Grading (DataNeedGrading)
	AssignmentGroups []AssignmentGroupSnap
	GradingScale     string

	// Enrollments (DataNeedEnrollments) — role counts + privacy-safe people stubs
	EnrollmentCounts   map[string]int
	People             []PersonSnap
	PendingInvitations []PendingInviteSnap

	// Feed (DataNeedFeed)
	FeedChannels         []FeedChannelSnap
	AnnouncementsWelcome *WelcomeMessageSnap

	// Files (DataNeedFiles)
	Files []FileSnap

	// Sections (DataNeedSections)
	Sections []SectionSnap

	// Accommodations (DataNeedAccommodations)
	AccommodationCount int

	// Standards (DataNeedStandards)
	StandardsCount         int
	StandardAlignedItemIDs map[uuid.UUID]struct{}

	// Content tools (DataNeedContentTools)
	ContentToolItemIDs map[uuid.UUID]struct{}

	// Module gating graph (DataNeedModulePrerequisites)
	ModulePrerequisiteEdges []PrerequisiteEdge
	HasItemCompletionRules  bool

	// Lazy payload filled by LazyLoaders during Evaluate (never by LoadSnapshot).
	Lazy map[LazyLoaderID]any

	// QueryCount is set by LoadSnapshot when a query counter is attached (tests).
	QueryCount int
}

// CourseFeatures mirrors the feature switches needed by Applies predicates.
type CourseFeatures struct {
	NotebookEnabled           bool
	FeedEnabled               bool
	CalendarEnabled           bool
	DiscussionsEnabled        bool
	FilesEnabled              bool
	AttendanceEnabled         bool
	StandardsAlignmentEnabled bool
	AdaptivePathsEnabled      bool
	ContentToolsEnabled       bool
	InteractiveQuizzesEnabled bool
	RequireCaptions           bool
	GroupSpacesEnabled        bool
	VisualBoardsEnabled       bool
	AiTutorEnabled            bool
	ModulesAiAssistantEnabled bool
}

// StructureItem is a course structure row subset for checklist rules.
type StructureItem struct {
	ID                uuid.UUID
	Kind              string
	Title             string
	ParentID          *uuid.UUID
	Published         bool
	VisibleFrom       *time.Time
	DueAt             *time.Time
	AssignmentGroupID *uuid.UUID
	Archived          bool
	SortOrder         int
}

// ItemMeta is lightweight per-item detail used by structure/assessment rules.
type ItemMeta struct {
	Kind                 string
	HasBody              bool
	PointsWorth          *int
	ExternalURL          string
	QuestionCount        int
	LateSubmissionPolicy string // "" when N/A (non-gradable kinds)
	Attribution          string // external_link attribution_text or library/textbook source
	BodyMarkdown         string // capped markdown for file-ref scanning (content pages)
	EmbeddedFileIDs      []uuid.UUID
}

// SyllabusSectionSnap is one syllabus section.
type SyllabusSectionSnap struct {
	Key      string
	Title    string
	HasBody  bool
	Markdown string
}

// OutcomeSnap is a learning outcome.
type OutcomeSnap struct {
	ID          uuid.UUID
	Title       string
	Description string
	SortOrder   int
}

// OutcomeLinkSnap links an outcome to a structure item.
type OutcomeLinkSnap struct {
	OutcomeID        uuid.UUID
	ItemID           uuid.UUID
	TargetKind       string
	MeasurementLevel string
	ItemTitle        string
	ItemKind         string
}

// GradableItem is a non-archived assignment/quiz used by alignment rules (FR-26).
type GradableItem struct {
	ID        uuid.UUID
	Title     string
	Kind      string
	ParentID  *uuid.UUID
	SortOrder int
	Points    *int
}

// PrerequisiteEdge is one module_id → prerequisite_module_id edge.
type PrerequisiteEdge struct {
	ModuleID             uuid.UUID
	PrerequisiteModuleID uuid.UUID
}

// AssignmentGroupSnap is an assignment group.
type AssignmentGroupSnap struct {
	ID     uuid.UUID
	Name   string
	Weight *float64
}

// PersonSnap is a privacy-safe enrollment stub (display name + opaque ID only).
type PersonSnap struct {
	UserID            uuid.UUID
	DisplayName       string
	Role              string
	InvitationPending bool
	EnrolledAt        *time.Time
	SectionID         *uuid.UUID
	Active            bool
	HasGuardianLink   bool
}

// PendingInviteSnap is a privacy-safe pending invitation (no email).
type PendingInviteSnap struct {
	DisplayName string
	UserID      uuid.UUID
	CreatedAt   time.Time
	DaysPending int
}

// FeedChannelSnap is a feed channel with optional latest root message time.
type FeedChannelSnap struct {
	ID          uuid.UUID
	Name        string
	LatestAt    *time.Time
	LatestTitle string
}

// WelcomeMessageSnap describes a staff welcome post on announcements.
type WelcomeMessageSnap struct {
	BodyLen       int
	AuthorIsStaff bool
	PostedAt      *time.Time
}

// FileSnap is course file metadata.
type FileSnap struct {
	ID          uuid.UUID
	DisplayName string
	ContentType string
	ByteSize    int64
}

// SectionSnap is a course section.
type SectionSnap struct {
	ID          uuid.UUID
	SectionCode string
	Name        string
	Status      string
}
