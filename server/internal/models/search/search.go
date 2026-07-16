// Package search holds JSON for GET /api/v1/search (server/src/models/search.rs).
package search

import "github.com/google/uuid"

// CourseItem is a catalog row in the search index.
type CourseItem struct {
	CourseCode                    string `json:"courseCode"`
	Title                         string `json:"title"`
	NotebookEnabled               bool   `json:"notebookEnabled"`
	FeedEnabled                   bool   `json:"feedEnabled"`
	CalendarEnabled               bool   `json:"calendarEnabled"`
	QuestionBankEnabled           bool   `json:"questionBankEnabled"`
	LockdownModeEnabled           bool   `json:"lockdownModeEnabled"`
	StandardsAlignmentEnabled     bool   `json:"standardsAlignmentEnabled"`
	AdaptivePathsEnabled          bool   `json:"adaptivePathsEnabled"`
	SRSEnabled                    bool   `json:"srsEnabled"`
	DiagnosticAssessmentsEnabled  bool   `json:"diagnosticAssessmentsEnabled"`
	HintScaffoldingEnabled        bool   `json:"hintScaffoldingEnabled"`
	MisconceptionDetectionEnabled bool   `json:"misconceptionDetectionEnabled"`
	DiscussionsEnabled            bool   `json:"discussionsEnabled"`
	CollabDocsEnabled             bool   `json:"collabDocsEnabled"`
	SbgEnabled                    bool   `json:"sbgEnabled"`
	LiveSessionsEnabled           bool   `json:"liveSessionsEnabled"`
	GroupSpacesEnabled            bool   `json:"groupSpacesEnabled"`
	OfficeHoursEnabled            bool   `json:"officeHoursEnabled"`
	FilesEnabled                  bool   `json:"filesEnabled"`
	AttendanceEnabled             bool   `json:"attendanceEnabled"`
	WhiteboardEnabled             bool   `json:"whiteboardEnabled"`
	ReportCardsEnabled            bool   `json:"reportCardsEnabled"`
	VisualBoardsEnabled           bool   `json:"visualBoardsEnabled"`
	InteractiveQuizzesEnabled     bool   `json:"interactiveQuizzesEnabled"`
}

// PersonItem is a roster person visible to the caller (when they have enrollments:read for that course).
type PersonItem struct {
	UserID      uuid.UUID `json:"userId"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"displayName"`
	Role        string    `json:"role"`
	CourseCode  string    `json:"courseCode"`
	CourseTitle string    `json:"courseTitle"`
}

// IndexResponse is the top-level /api/v1/search payload.
type IndexResponse struct {
	Courses []CourseItem `json:"courses"`
	People  []PersonItem `json:"people"`
}

// QueryResultItem is one row in GET /api/v1/search/query grouped results.
type QueryResultItem struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Path     string  `json:"path"`
	Score    float64 `json:"score,omitempty"`
}

// QueryGroup is a capped result bucket (courses, people, content, …).
type QueryGroup struct {
	Type  string            `json:"type"`
	Label string            `json:"label"`
	Total int               `json:"total"`
	Items []QueryResultItem `json:"items"`
}

// QueryResponse is the top-level /api/v1/search/query payload.
type QueryResponse struct {
	Groups []QueryGroup `json:"groups"`
	TookMs int64        `json:"tookMs"`
}
