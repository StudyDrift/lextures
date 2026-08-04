package coursechecklist

import (
	"time"

	"github.com/google/uuid"
)

// CourseSnapshot is the in-memory course view evaluators receive (FR-6).
// Fields are populated according to the DataNeeds passed to LoadSnapshot.
type CourseSnapshot struct {
	CourseCode string
	CourseID   uuid.UUID

	// Course (DataNeedCourse)
	Title           string
	Published       bool
	StartsAt        *time.Time
	EndsAt          *time.Time
	CourseTimezone  *string
	ScheduleMode    string
	SectionsEnabled bool
	FeedEnabled     bool
	FilesEnabled    bool
	SbgEnabled      bool
	StandardsEnabled bool
	CourseType      string
	CourseMode      string
	Features        CourseFeatures

	// Structure (DataNeedStructure)
	StructureItems []StructureItem

	// Item detail metadata (DataNeedItemMeta)
	ItemMeta map[uuid.UUID]ItemMeta

	// Syllabus (DataNeedSyllabus)
	SyllabusSections []SyllabusSectionSnap

	// Outcomes (DataNeedOutcomes)
	Outcomes     []OutcomeSnap
	OutcomeLinks []OutcomeLinkSnap

	// Grading (DataNeedGrading)
	AssignmentGroups []AssignmentGroupSnap
	GradingScale     string

	// Enrollments (DataNeedEnrollments) — role counts + privacy-safe people stubs
	EnrollmentCounts map[string]int
	People           []PersonSnap

	// Feed (DataNeedFeed)
	FeedChannels []FeedChannelSnap

	// Files (DataNeedFiles)
	Files []FileSnap

	// Sections (DataNeedSections)
	Sections []SectionSnap

	// Accommodations (DataNeedAccommodations)
	AccommodationCount int

	// Standards (DataNeedStandards)
	StandardsCount int

	// Lazy payload filled by LazyLoaders during Evaluate (never by LoadSnapshot).
	Lazy map[LazyLoaderID]any

	// QueryCount is set by LoadSnapshot when a query counter is attached (tests).
	QueryCount int
}

// CourseFeatures mirrors the feature switches needed by Applies predicates.
type CourseFeatures struct {
	NotebookEnabled              bool
	FeedEnabled                  bool
	CalendarEnabled              bool
	DiscussionsEnabled           bool
	FilesEnabled                 bool
	AttendanceEnabled            bool
	StandardsAlignmentEnabled    bool
	AdaptivePathsEnabled         bool
	ContentToolsEnabled          bool
	InteractiveQuizzesEnabled    bool
	RequireCaptions              bool
}

// StructureItem is a course structure row subset for checklist rules.
type StructureItem struct {
	ID                uuid.UUID
	Kind              string
	Title             string
	ParentID          *uuid.UUID
	Published         bool
	DueAt             *time.Time
	AssignmentGroupID *uuid.UUID
	Archived          bool
}

// ItemMeta is lightweight per-item detail used by structure/assessment rules.
type ItemMeta struct {
	Kind          string
	HasBody       bool
	PointsWorth   *int
	ExternalURL   string
	QuestionCount int
}

// SyllabusSectionSnap is one syllabus section.
type SyllabusSectionSnap struct {
	Key     string
	Title   string
	HasBody bool
}

// OutcomeSnap is a learning outcome.
type OutcomeSnap struct {
	ID    uuid.UUID
	Title string
}

// OutcomeLinkSnap links an outcome to a structure item.
type OutcomeLinkSnap struct {
	OutcomeID uuid.UUID
	ItemID    uuid.UUID
}

// AssignmentGroupSnap is an assignment group.
type AssignmentGroupSnap struct {
	ID    uuid.UUID
	Name  string
	Weight *float64
}

// PersonSnap is a privacy-safe enrollment stub (display name + opaque ID only).
type PersonSnap struct {
	UserID      uuid.UUID
	DisplayName string
	Role        string
}

// FeedChannelSnap is a feed channel with optional latest root message time.
type FeedChannelSnap struct {
	ID          uuid.UUID
	Name        string
	LatestAt    *time.Time
	LatestTitle string
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
