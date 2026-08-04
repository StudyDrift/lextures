package courseexportimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/models/coursesyllabus"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

const exportFormatVersion int32 = 1

// Bundle is the JSON body for GET /api/v1/courses/{course_code}/export.
// Structure and grading use the same shapes as the live LMS APIs.
type Bundle struct {
	FormatVersion             int32                                         `json:"formatVersion"`
	ExportedAt                time.Time                                     `json:"exportedAt"`
	CourseCode                string                                        `json:"courseCode"`
	Course                    courseexport.CourseExportSnapshot             `json:"course"`
	Syllabus                  []coursesyllabus.SyllabusSection              `json:"syllabus"`
	RequireSyllabusAcceptance bool                                          `json:"requireSyllabusAcceptance"`
	Grading                   coursegrading.SettingsResponse                `json:"grading"`
	Structure                 []coursestructure.ItemResponse                `json:"structure"`
	ContentPages              map[uuid.UUID]courseexport.ExportedContentPageBody `json:"contentPages"`
	Assignments               map[uuid.UUID]courseexport.ExportedAssignmentBody  `json:"assignments"`
	Quizzes                   map[uuid.UUID]courseexport.ExportedQuizBody        `json:"quizzes"`
	Enrollments               []courseexport.ExportedCourseEnrollment            `json:"enrollments"`
	// Adaptive content authoring only (AC.1 FR-9) — never profiles/variants/servings.
	AdaptiveContentSettings *courseexport.ExportedAdaptiveContentSettings `json:"adaptiveContentSettings,omitempty"`
	AdaptiveContentUnits    []courseexport.ExportedAdaptiveContentUnit    `json:"adaptiveContentUnits,omitempty"`
	// Content tool authoring only — never learner state/events/grade links.
	ContentToolSettings  *courseexport.ExportedContentToolSettings  `json:"contentToolSettings,omitempty"`
	ContentToolInstances []courseexport.ExportedContentToolInstance `json:"contentToolInstances,omitempty"`
	// Embedded content images (course.course_files) with base64 bodies.
	CourseFiles []courseexport.ExportedCourseFile `json:"courseFiles,omitempty"`
	// File manager hierarchy with base64 bodies on each item.
	FileFolders []courseexport.ExportedFileFolder `json:"fileFolders,omitempty"`
	FileItems   []courseexport.ExportedFileItem   `json:"fileItems,omitempty"`
}

// BlobOptions configures how export/import reads and writes course file blobs.
// When Storage is nil, FilesRoot (or "data/course-files") is used for local disk I/O.
type BlobOptions struct {
	FilesRoot string
	Storage   filestorage.Driver
}

// ErrNotFound is returned when the course code does not exist.
var ErrNotFound = errors.New("course not found")

// BuildExport builds a full course JSON export bundle (Rust `build_export`).
func BuildExport(ctx context.Context, pool *pgxpool.Pool, courseCode string, blobOpts BlobOptions) (*Bundle, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	courseCode = strings.TrimSpace(courseCode)
	if courseCode == "" {
		return nil, ErrNotFound
	}

	pub, err := course.GetPublicByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return nil, err
	}
	if pub == nil {
		return nil, ErrNotFound
	}
	courseID, err := uuid.Parse(pub.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid course id: %w", err)
	}

	snap := courseexport.CourseExportSnapshot{
		Title:                         pub.Title,
		Description:                   pub.Description,
		HeroImageURL:                  pub.HeroImageURL,
		HeroImageObjectPosition:       pub.HeroImageObjectPosition,
		StartsAt:                      pub.StartsAt,
		EndsAt:                        pub.EndsAt,
		VisibleFrom:                   pub.VisibleFrom,
		HiddenAt:                      pub.HiddenAt,
		ScheduleMode:                  orDefault(pub.ScheduleMode, "fixed"),
		RelativeEndAfter:              pub.RelativeEndAfter,
		RelativeHiddenAfter:           pub.RelativeHiddenAfter,
		RelativeScheduleAnchorAt:      pub.RelativeScheduleAnchorAt,
		Published:                     pub.Published,
		MarkdownThemePreset:           pub.MarkdownThemePreset,
		NotebookEnabled:               pub.NotebookEnabled,
		FeedEnabled:                   pub.FeedEnabled,
		CalendarEnabled:               pub.CalendarEnabled,
		QuestionBankEnabled:           pub.QuestionBankEnabled,
		LockdownModeEnabled:           pub.LockdownModeEnabled,
		StandardsAlignmentEnabled:     pub.StandardsAlignmentEnabled,
		AdaptivePathsEnabled:          pub.AdaptivePathsEnabled,
		SrsEnabled:                    pub.SRSEnabled,
		DiagnosticAssessmentsEnabled:  pub.DiagnosticAssessmentsEnabled,
		HintScaffoldingEnabled:        pub.HintScaffoldingEnabled,
		MisconceptionDetectionEnabled: pub.MisconceptionDetectionEnabled,
		AdaptiveContentEnabled:        pub.AdaptiveContentEnabled,
		ContentToolsEnabled:           pub.ContentToolsEnabled,
		CourseType:                    orDefault(pub.CourseType, "traditional"),
	}
	if pub.MarkdownThemeCustom != nil {
		snap.MarkdownThemeCustom = append(json.RawMessage(nil), *pub.MarkdownThemeCustom...)
	}

	grading, err := coursegrading.GetSettingsForCourseCode(ctx, pool, courseCode)
	if err != nil {
		return nil, err
	}
	if grading == nil {
		return nil, ErrNotFound
	}
	if grading.AssignmentGroups == nil {
		grading.AssignmentGroups = []coursegrading.AssignmentGroupPublic{}
	}

	syllabusPayload, err := course.GetSyllabusByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return nil, err
	}
	syllabus := []coursesyllabus.SyllabusSection{}
	requireAcceptance := false
	if syllabusPayload != nil {
		requireAcceptance = syllabusPayload.RequireSyllabusAcceptance
		for _, s := range syllabusPayload.Sections {
			syllabus = append(syllabus, coursesyllabus.SyllabusSection{
				ID:       s.ID,
				Heading:  s.Heading,
				Markdown: s.Markdown,
			})
		}
	}

	structure, err := coursestructure.ListForCourseWithEnrichment(ctx, pool, courseID, true)
	if err != nil {
		return nil, err
	}
	if structure == nil {
		structure = []coursestructure.ItemResponse{}
	}

	contentPages := map[uuid.UUID]courseexport.ExportedContentPageBody{}
	assignments := map[uuid.UUID]courseexport.ExportedAssignmentBody{}
	quizzes := map[uuid.UUID]courseexport.ExportedQuizBody{}

	for i := range structure {
		it := &structure[i]
		itemID, err := uuid.Parse(it.ID)
		if err != nil {
			continue
		}
		switch it.Kind {
		case "content_page":
			row, err := coursemodulecontent.GetForCourseItem(ctx, pool, courseID, itemID)
			if err != nil {
				return nil, err
			}
			if row != nil {
				contentPages[itemID] = courseexport.ExportedContentPageBody{
					Markdown: row.Markdown,
					DueAt:    row.DueAt,
				}
			}
		case "assignment":
			row, err := coursemoduleassignments.GetForCourseItem(ctx, pool, courseID, itemID)
			if err != nil {
				return nil, err
			}
			if row != nil {
				rubric := json.RawMessage(nil)
				if len(row.RubricJSON) > 0 {
					rubric = append(json.RawMessage(nil), row.RubricJSON...)
				}
				assignments[itemID] = courseexport.ExportedAssignmentBody{
					Markdown:                     row.Markdown,
					DueAt:                        row.DueAt,
					PointsWorth:                  intPtrToI32(row.PointsWorth),
					AvailableFrom:                row.AvailableFrom,
					AvailableUntil:               row.AvailableUntil,
					AssignmentAccessCode:         row.AssignmentAccessCode,
					SubmissionAllowText:          row.SubmissionAllowText,
					SubmissionAllowFileUpload:    row.SubmissionAllowFileUpload,
					SubmissionAllowURL:           row.SubmissionAllowURL,
					LateSubmissionPolicy:         orDefault(row.LateSubmissionPolicy, "allow"),
					LatePenaltyPercent:           intPtrToI32(row.LatePenaltyPercent),
					Rubric:                       rubric,
					BlindGrading:                 row.BlindGrading,
					OriginalityDetection:         orDefault(row.OriginalityDetection, "disabled"),
					OriginalityStudentVisibility: orDefault(row.OriginalityStudentVisibility, "hide"),
					GradingType:                  row.GradingType,
				}
			}
		case "quiz":
			row, err := coursemodulequizzes.GetForCourseItem(ctx, pool, courseID, itemID)
			if err != nil {
				return nil, err
			}
			if row != nil {
				srcIDs := row.AdaptiveSourceItemIDs
				if srcIDs == nil {
					srcIDs = []uuid.UUID{}
				}
				quizzes[itemID] = courseexport.ExportedQuizBody{
					Markdown:                    row.Markdown,
					DueAt:                       row.DueAt,
					AvailableFrom:               row.AvailableFrom,
					AvailableUntil:              row.AvailableUntil,
					UnlimitedAttempts:           row.UnlimitedAttempts,
					MaxAttempts:                 row.MaxAttempts,
					GradeAttemptPolicy:          orDefault(row.GradeAttemptPolicy, "latest"),
					PassingScorePercent:         row.PassingScorePercent,
					PointsWorth:                 row.PointsWorth,
					LateSubmissionPolicy:        orDefault(row.LateSubmissionPolicy, "allow"),
					LatePenaltyPercent:          row.LatePenaltyPercent,
					TimeLimitMinutes:            row.TimeLimitMinutes,
					TimerPauseWhenTabHidden:     row.TimerPauseWhenTabHidden,
					PerQuestionTimeLimitSeconds: row.PerQuestionTimeLimitSeconds,
					ShowScoreTiming:             orDefault(row.ShowScoreTiming, "immediate"),
					ReviewVisibility:            orDefault(row.ReviewVisibility, "full"),
					ReviewWhen:                  orDefault(row.ReviewWhen, "always"),
					OneQuestionAtATime:          row.OneQuestionAtATime,
					ShuffleQuestions:            row.ShuffleQuestions,
					ShuffleChoices:              row.ShuffleChoices,
					AllowBackNavigation:         row.AllowBackNavigation,
					LockdownMode:                orDefault(row.LockdownMode, "standard"),
					FocusLossThreshold:          row.FocusLossThreshold,
					QuizAccessCode:              row.QuizAccessCode,
					AdaptiveDifficulty:          orDefault(row.AdaptiveDifficulty, "standard"),
					AdaptiveTopicBalance:        row.AdaptiveTopicBalance,
					AdaptiveStopRule:            orDefault(row.AdaptiveStopRule, "fixed_count"),
					RandomQuestionPoolCount:     row.RandomQuestionPoolCount,
					Questions:                   row.Questions,
					IsAdaptive:                  row.IsAdaptive,
					AdaptiveSystemPrompt:        row.AdaptiveSystemPrompt,
					AdaptiveSourceItemIDs:       srcIDs,
					AdaptiveQuestionCount:       row.AdaptiveQuestionCount,
					AdaptiveDeliveryMode:        orDefault(row.AdaptiveDeliveryMode, "ai"),
				}
			}
		}
	}

	emailRoles, err := enrollment.ListEmailRolesForCourseExport(ctx, pool, courseCode)
	if err != nil {
		return nil, err
	}
	enrollments := make([]courseexport.ExportedCourseEnrollment, 0, len(emailRoles))
	for _, er := range emailRoles {
		enrollments = append(enrollments, courseexport.ExportedCourseEnrollment{
			Email:       er.Email,
			Role:        er.Role,
			DisplayName: er.DisplayName,
		})
	}

	bundle := &Bundle{
		FormatVersion:             exportFormatVersion,
		ExportedAt:                time.Now().UTC(),
		CourseCode:                pub.CourseCode,
		Course:                    snap,
		Syllabus:                  syllabus,
		RequireSyllabusAcceptance: requireAcceptance,
		Grading:                   *grading,
		Structure:                 structure,
		ContentPages:              contentPages,
		Assignments:               assignments,
		Quizzes:                   quizzes,
		Enrollments:               enrollments,
	}

	// Optional ACE authoring data (settings + units only).
	if settings, err := acrepo.GetSettings(ctx, pool, courseID); err != nil {
		return nil, err
	} else if settings != nil {
		axes := settings.AllowedAxes
		if axes == nil {
			axes = []string{}
		}
		bundle.AdaptiveContentSettings = &courseexport.ExportedAdaptiveContentSettings{
			AllowedAxes:               axes,
			DefaultStrategy:           settings.DefaultStrategy,
			HoldoutPercent:            settings.HoldoutPercent,
			MonthlyTokenBudget:        settings.MonthlyTokenBudget,
			RequireInstructorApproval: settings.RequireInstructorApproval,
			StudentOptoutAllowed:      settings.StudentOptoutAllowed,
		}
	}
	if units, err := acrepo.ListUnits(ctx, pool, courseID); err != nil {
		return nil, err
	} else if len(units) > 0 {
		out := make([]courseexport.ExportedAdaptiveContentUnit, 0, len(units))
		for _, u := range units {
			axes := u.AllowedAxes
			if axes == nil {
				axes = []string{}
			}
			out = append(out, courseexport.ExportedAdaptiveContentUnit{
				TargetKind:           u.TargetKind,
				TargetModuleItemID:   u.TargetModuleItemID,
				TargetOutcomeID:      u.TargetOutcomeID,
				BaseContentItemID:    u.BaseContentItemID,
				PreAssessmentItemID:  u.PreAssessmentItemID,
				PostAssessmentItemID: u.PostAssessmentItemID,
				AllowedAxes:          axes,
				Status:               u.Status,
				TriggerMode:          u.TriggerMode,
				MasteryFreshnessDays: u.MasteryFreshnessDays,
			})
		}
		bundle.AdaptiveContentUnits = out
	}

	// Content Tools authoring (settings + instances). Markdown bodies already embed
	// ```lex-tool fences by instance id — keep those ids stable on import.
	if settings, err := contenttools.GetSettings(ctx, pool, courseID); err != nil {
		return nil, err
	} else if settings != nil {
		allowed := settings.AllowedToolIDs
		if allowed == nil {
			allowed = []string{}
		}
		allowlist := settings.LinkHostAllowlist
		if allowlist == nil {
			allowlist = []string{}
		}
		bundle.ContentToolSettings = &courseexport.ExportedContentToolSettings{
			AllowedToolIDs:       allowed,
			StudentResetAllowed:  settings.StudentResetAllowed,
			MaxInstancesPerItem:  settings.MaxInstancesPerItem,
			MonthlyAITokenBudget: settings.MonthlyAITokenBudget,
			DailyAICallsPerUser:  settings.DailyAICallsPerUser,
			LinkIngestionMode:    settings.LinkIngestionMode,
			LinkHostAllowlist:    allowlist,
			GradeLinksAllowed:    settings.GradeLinksAllowed,
		}
	}
	if instances, err := contenttools.ListInstances(ctx, pool, courseID, nil, "", true); err != nil {
		return nil, err
	} else if len(instances) > 0 {
		out := make([]courseexport.ExportedContentToolInstance, 0, len(instances))
		for _, in := range instances {
			cfg := in.ConfigJSON
			if len(cfg) == 0 {
				cfg = json.RawMessage(`{}`)
			} else {
				cfg = append(json.RawMessage(nil), cfg...)
			}
			out = append(out, courseexport.ExportedContentToolInstance{
				ID:                  in.ID,
				StructureItemID:     in.StructureItemID,
				HostKind:            in.HostKind,
				SectionKey:          in.SectionKey,
				ToolID:              in.ToolID,
				ToolVersion:         in.ToolVersion,
				Title:               in.Title,
				ConfigJSON:          cfg,
				ConfigSchemaVersion: in.ConfigSchemaVersion,
				Status:              in.Status,
			})
		}
		bundle.ContentToolInstances = out
	}

	if err := attachCourseFilesToExport(ctx, pool, courseID, courseCode, blobOpts, bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func intPtrToI32(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v)
	return &n
}
