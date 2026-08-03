package courseexport

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/coursegrading"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/models/coursestructure"
	"github.com/lextures/lextures/server/internal/models/coursesyllabus"
)

type CourseImportMode string

const (
	CourseImportModeErase     CourseImportMode = "erase"
	CourseImportModeMergeAdd  CourseImportMode = "mergeAdd"
	CourseImportModeOverwrite CourseImportMode = "overwrite"
)

type CourseExportSnapshot struct {
	Title                         string          `json:"title"`
	Description                   string          `json:"description"`
	HeroImageURL                  *string         `json:"heroImageUrl"`
	HeroImageObjectPosition       *string         `json:"heroImageObjectPosition"`
	StartsAt                      *time.Time      `json:"startsAt"`
	EndsAt                        *time.Time      `json:"endsAt"`
	VisibleFrom                   *time.Time      `json:"visibleFrom"`
	HiddenAt                      *time.Time      `json:"hiddenAt"`
	ScheduleMode                  string          `json:"scheduleMode"`
	RelativeEndAfter              *string         `json:"relativeEndAfter"`
	RelativeHiddenAfter           *string         `json:"relativeHiddenAfter"`
	RelativeScheduleAnchorAt      *time.Time      `json:"relativeScheduleAnchorAt"`
	Published                     bool            `json:"published"`
	MarkdownThemePreset           string          `json:"markdownThemePreset"`
	MarkdownThemeCustom           json.RawMessage `json:"markdownThemeCustom"`
	NotebookEnabled               bool            `json:"notebookEnabled"`
	FeedEnabled                   bool            `json:"feedEnabled"`
	CalendarEnabled               bool            `json:"calendarEnabled"`
	QuestionBankEnabled           bool            `json:"questionBankEnabled"`
	LockdownModeEnabled           bool            `json:"lockdownModeEnabled"`
	StandardsAlignmentEnabled     bool            `json:"standardsAlignmentEnabled"`
	AdaptivePathsEnabled          bool            `json:"adaptivePathsEnabled"`
	SrsEnabled                    bool            `json:"srsEnabled"`
	DiagnosticAssessmentsEnabled  bool            `json:"diagnosticAssessmentsEnabled"`
	HintScaffoldingEnabled        bool            `json:"hintScaffoldingEnabled"`
	MisconceptionDetectionEnabled bool            `json:"misconceptionDetectionEnabled"`
	AdaptiveContentEnabled        bool            `json:"adaptiveContentEnabled"`
	ContentToolsEnabled           bool            `json:"contentToolsEnabled"`
	CourseType                    string          `json:"courseType"`
}

// ExportedAdaptiveContentSettings is course ACE config for export/import (plan AC.1 FR-9).
// Per-student profiles/variants/servings are never exported.
type ExportedAdaptiveContentSettings struct {
	AllowedAxes               []string `json:"allowedAxes"`
	DefaultStrategy           string   `json:"defaultStrategy"`
	HoldoutPercent            int16    `json:"holdoutPercent"`
	MonthlyTokenBudget        int64    `json:"monthlyTokenBudget"`
	RequireInstructorApproval bool     `json:"requireInstructorApproval"`
	StudentOptoutAllowed      bool     `json:"studentOptoutAllowed"`
}

// ExportedAdaptiveContentUnit is an ACE unit for course duplication (plan AC.1 FR-9 / AC.2).
type ExportedAdaptiveContentUnit struct {
	TargetKind           string     `json:"targetKind"`
	TargetModuleItemID   *uuid.UUID `json:"targetModuleItemId,omitempty"`
	TargetOutcomeID      *uuid.UUID `json:"targetOutcomeId,omitempty"`
	BaseContentItemID    uuid.UUID  `json:"baseContentItemId"`
	PreAssessmentItemID  *uuid.UUID `json:"preAssessmentItemId,omitempty"`
	PostAssessmentItemID *uuid.UUID `json:"postAssessmentItemId,omitempty"`
	AllowedAxes          []string   `json:"allowedAxes"`
	Status               string     `json:"status"`
	TriggerMode          string     `json:"triggerMode,omitempty"`
	MasteryFreshnessDays int16      `json:"masteryFreshnessDays,omitempty"`
}

// ExportedContentToolSettings is per-course Content Tools config for export/import.
// Learner state, events, and grade links are never exported.
type ExportedContentToolSettings struct {
	AllowedToolIDs       []string `json:"allowedToolIds"`
	StudentResetAllowed  bool     `json:"studentResetAllowed"`
	MaxInstancesPerItem  int16    `json:"maxInstancesPerItem"`
	MonthlyAITokenBudget int64    `json:"monthlyAiTokenBudget"`
	DailyAICallsPerUser  int      `json:"dailyAiCallsPerUser"`
	LinkIngestionMode    string   `json:"linkIngestionMode"`
	LinkHostAllowlist    []string `json:"linkHostAllowlist"`
	GradeLinksAllowed    bool     `json:"gradeLinksAllowed"`
}

// ExportedContentToolInstance is a placed tool + config (no learner state).
// Instance IDs are preserved so ```lex-tool fences in markdown keep resolving.
type ExportedContentToolInstance struct {
	ID                  uuid.UUID       `json:"id"`
	StructureItemID     *uuid.UUID      `json:"structureItemId,omitempty"`
	HostKind            string          `json:"hostKind"`
	SectionKey          *string         `json:"sectionKey,omitempty"`
	ToolID              string          `json:"toolId"`
	ToolVersion         string          `json:"toolVersion"`
	Title               *string         `json:"title,omitempty"`
	ConfigJSON          json.RawMessage `json:"configJson"`
	ConfigSchemaVersion int             `json:"configSchemaVersion"`
	Status              string          `json:"status"`
}

// ExportedCourseFile is an embedded content asset (course.course_files) with base64 body.
// IDs are preserved so markdown / hero URLs of the form
// /api/v1/courses/{code}/course-files/{id}/content keep resolving after import.
type ExportedCourseFile struct {
	ID               uuid.UUID `json:"id"`
	OriginalFilename string    `json:"originalFilename"`
	MimeType         string    `json:"mimeType"`
	ByteSize         int64     `json:"byteSize"`
	// ContentBase64 is the raw file bytes (standard base64). Omitted when the blob
	// was missing at export time.
	ContentBase64 string `json:"contentBase64,omitempty"`
}

// ExportedFileFolder is a course file-manager folder (course.file_folders).
type ExportedFileFolder struct {
	ID       uuid.UUID  `json:"id"`
	ParentID *uuid.UUID `json:"parentId"`
	Name     string     `json:"name"`
}

// ExportedFileItem is a course file-manager file (course.file_items) with base64 body.
// IDs are preserved so /api/v1/courses/{code}/files/items/{id}/content URLs keep resolving.
type ExportedFileItem struct {
	ID               uuid.UUID  `json:"id"`
	FolderID         *uuid.UUID `json:"folderId"`
	OriginalFilename string     `json:"originalFilename"`
	DisplayName      string     `json:"displayName"`
	MimeType         string     `json:"mimeType"`
	ByteSize         int64      `json:"byteSize"`
	ContentBase64    string     `json:"contentBase64,omitempty"`
}

type ExportedContentPageBody struct {
	Markdown string     `json:"markdown"`
	DueAt    *time.Time `json:"dueAt"`
}

type ExportedAssignmentBody struct {
	Markdown                     string          `json:"markdown"`
	DueAt                        *time.Time      `json:"dueAt"`
	PointsWorth                  *int32          `json:"pointsWorth"`
	AvailableFrom                *time.Time      `json:"availableFrom"`
	AvailableUntil               *time.Time      `json:"availableUntil"`
	AssignmentAccessCode         *string         `json:"assignmentAccessCode"`
	SubmissionAllowText          bool            `json:"submissionAllowText"`
	SubmissionAllowFileUpload    bool            `json:"submissionAllowFileUpload"`
	SubmissionAllowURL           bool            `json:"submissionAllowUrl"`
	LateSubmissionPolicy         string          `json:"lateSubmissionPolicy"`
	LatePenaltyPercent           *int32          `json:"latePenaltyPercent"`
	Rubric                       json.RawMessage `json:"rubric"`
	BlindGrading                 bool            `json:"blindGrading"`
	OriginalityDetection         string          `json:"originalityDetection"`
	OriginalityStudentVisibility string          `json:"originalityStudentVisibility"`
	GradingType                  *string         `json:"gradingType"`
}

type ExportedQuizBody struct {
	Markdown                    string                            `json:"markdown"`
	DueAt                       *time.Time                        `json:"dueAt"`
	AvailableFrom               *time.Time                        `json:"availableFrom"`
	AvailableUntil              *time.Time                        `json:"availableUntil"`
	UnlimitedAttempts           bool                              `json:"unlimitedAttempts"`
	MaxAttempts                 int32                             `json:"maxAttempts"`
	GradeAttemptPolicy          string                            `json:"gradeAttemptPolicy"`
	PassingScorePercent         *int32                            `json:"passingScorePercent"`
	PointsWorth                 *int32                            `json:"pointsWorth"`
	LateSubmissionPolicy        string                            `json:"lateSubmissionPolicy"`
	LatePenaltyPercent          *int32                            `json:"latePenaltyPercent"`
	TimeLimitMinutes            *int32                            `json:"timeLimitMinutes"`
	TimerPauseWhenTabHidden     bool                              `json:"timerPauseWhenTabHidden"`
	PerQuestionTimeLimitSeconds *int32                            `json:"perQuestionTimeLimitSeconds"`
	ShowScoreTiming             string                            `json:"showScoreTiming"`
	ReviewVisibility            string                            `json:"reviewVisibility"`
	ReviewWhen                  string                            `json:"reviewWhen"`
	OneQuestionAtATime          bool                              `json:"oneQuestionAtATime"`
	ShuffleQuestions            bool                              `json:"shuffleQuestions"`
	ShuffleChoices              bool                              `json:"shuffleChoices"`
	AllowBackNavigation         bool                              `json:"allowBackNavigation"`
	LockdownMode                string                            `json:"lockdownMode"`
	FocusLossThreshold          *int32                            `json:"focusLossThreshold"`
	QuizAccessCode              *string                           `json:"quizAccessCode"`
	AdaptiveDifficulty          string                            `json:"adaptiveDifficulty"`
	AdaptiveTopicBalance        bool                              `json:"adaptiveTopicBalance"`
	AdaptiveStopRule            string                            `json:"adaptiveStopRule"`
	RandomQuestionPoolCount     *int32                            `json:"randomQuestionPoolCount"`
	Questions                   []coursemodulequiz.QuizQuestion   `json:"questions"`
	IsAdaptive                  bool                              `json:"isAdaptive"`
	AdaptiveSystemPrompt        string                            `json:"adaptiveSystemPrompt"`
	AdaptiveSourceItemIDs       []uuid.UUID                       `json:"adaptiveSourceItemIds"`
	AdaptiveQuestionCount       int32                             `json:"adaptiveQuestionCount"`
	AdaptiveDeliveryMode        string                            `json:"adaptiveDeliveryMode"`
}

type ExportedCourseEnrollment struct {
	Email               string  `json:"email"`
	Role                string  `json:"role"`
	InstructorGrantRole *string `json:"instructorGrantRole"`
	DisplayName         *string `json:"displayName"`
}

type CourseExportV1 struct {
	FormatVersion             int32                                            `json:"formatVersion"`
	ExportedAt                time.Time                                        `json:"exportedAt"`
	CourseCode                string                                           `json:"courseCode"`
	Course                    CourseExportSnapshot                             `json:"course"`
	Syllabus                  []coursesyllabus.SyllabusSection                 `json:"syllabus"`
	RequireSyllabusAcceptance bool                                             `json:"requireSyllabusAcceptance"`
	Grading                   coursegrading.CourseGradingSettingsResponse      `json:"grading"`
	Structure                 []coursestructure.CourseStructureItemResponse    `json:"structure"`
	ContentPages              map[uuid.UUID]ExportedContentPageBody            `json:"contentPages"`
	Assignments               map[uuid.UUID]ExportedAssignmentBody             `json:"assignments"`
	Quizzes                   map[uuid.UUID]ExportedQuizBody                   `json:"quizzes"`
	Enrollments               []ExportedCourseEnrollment                       `json:"enrollments"`
	// AdaptiveContentSettings and AdaptiveContentUnits are optional ACE authoring data (AC.1).
	// Never includes profiles, variants, or servings.
	AdaptiveContentSettings *ExportedAdaptiveContentSettings `json:"adaptiveContentSettings,omitempty"`
	AdaptiveContentUnits    []ExportedAdaptiveContentUnit    `json:"adaptiveContentUnits,omitempty"`
	// Content tool authoring only — never learner state / events.
	ContentToolSettings  *ExportedContentToolSettings  `json:"contentToolSettings,omitempty"`
	ContentToolInstances []ExportedContentToolInstance `json:"contentToolInstances,omitempty"`
	// Embedded content images (course.course_files) with base64 bodies.
	CourseFiles []ExportedCourseFile `json:"courseFiles,omitempty"`
	// File manager hierarchy (course.file_folders + course.file_items).
	FileFolders []ExportedFileFolder `json:"fileFolders,omitempty"`
	FileItems   []ExportedFileItem   `json:"fileItems,omitempty"`
}

type CourseImportRequest struct {
	Mode   CourseImportMode `json:"mode"`
	Export CourseExportV1   `json:"export"`
}

type CanvasImportInclude struct {
	Modules       bool `json:"modules"`
	Assignments   bool `json:"assignments"`
	Quizzes       bool `json:"quizzes"`
	Enrollments   bool `json:"enrollments"`
	Grades        bool `json:"grades"`
	Settings      bool `json:"settings"`
	Files         bool `json:"files"`
	Announcements bool `json:"announcements"`
}

type CourseCanvasImportRequest struct {
	Mode          CourseImportMode   `json:"mode"`
	CanvasBaseURL string             `json:"canvasBaseUrl"`
	CanvasCourseID string            `json:"canvasCourseId"`
	AccessToken   string             `json:"accessToken"`
	Include       CanvasImportInclude `json:"include"`
}
