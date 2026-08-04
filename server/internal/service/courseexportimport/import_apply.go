package courseexportimport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/models/coursesyllabus"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleexternallinks"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

func applyGradingFromExport(ctx context.Context, pool *pgxpool.Pool, courseCode string, grading *coursegrading.SettingsResponse) error {
	if grading == nil {
		return nil
	}
	return coursegrading.ReplaceAssignmentGroupsForImport(ctx, pool, courseCode, strings.TrimSpace(grading.GradingScale), grading.AssignmentGroups)
}

func mergeAddGradingGroups(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, grading *coursegrading.SettingsResponse) error {
	if grading == nil {
		return nil
	}
	for _, g := range grading.AssignmentGroups {
		if _, err := coursegrading.InsertAssignmentGroupIfMissing(ctx, pool, courseID, g.ID, g.SortOrder, g.Name, g.WeightPercent); err != nil {
			return err
		}
	}
	return nil
}

func applyCourseSnapshot(ctx context.Context, pool *pgxpool.Pool, courseCode string, snap *courseexport.CourseExportSnapshot) error {
	if snap == nil {
		return InvalidInput("Export is missing course snapshot.")
	}
	cur, err := course.GetPublicByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	if cur == nil {
		return ErrNotFound
	}

	mode := strings.TrimSpace(snap.ScheduleMode)
	if mode != "relative" {
		mode = "fixed"
	}
	var startsAt, endsAt, visibleFrom, hiddenAt *time.Time
	var relativeEndAfter, relativeHiddenAfter *string
	var relativeAnchor *time.Time
	if mode == "relative" {
		anchor := snap.RelativeScheduleAnchorAt
		if anchor == nil {
			anchor = snap.StartsAt
		}
		if anchor == nil {
			now := time.Now().UTC()
			anchor = &now
		}
		relativeEndAfter = snap.RelativeEndAfter
		relativeHiddenAfter = snap.RelativeHiddenAfter
		relativeAnchor = anchor
	} else {
		startsAt = snap.StartsAt
		endsAt = snap.EndsAt
		visibleFrom = snap.VisibleFrom
		hiddenAt = snap.HiddenAt
	}

	homeLanding := cur.CourseHomeLanding
	if homeLanding == "" {
		homeLanding = "modules"
	}
	var homeItem *uuid.UUID
	if cur.CourseHomeContentItemID != nil && *cur.CourseHomeContentItemID != "" {
		if id, perr := uuid.Parse(*cur.CourseHomeContentItemID); perr == nil {
			homeItem = &id
		}
	}

	if _, err := course.UpdateCourse(ctx, pool, courseCode,
		strings.TrimSpace(snap.Title), strings.TrimSpace(snap.Description), snap.Published,
		startsAt, endsAt, visibleFrom, hiddenAt,
		mode, relativeEndAfter, relativeHiddenAfter, relativeAnchor,
		homeLanding, homeItem, cur.CourseTimezone,
	); err != nil {
		return err
	}

	preset := strings.TrimSpace(snap.MarkdownThemePreset)
	if preset == "" {
		preset = "classic"
	}
	var custom []byte
	if len(snap.MarkdownThemeCustom) > 0 {
		custom = append([]byte(nil), snap.MarkdownThemeCustom...)
	}
	if _, err := course.UpdateMarkdownTheme(ctx, pool, courseCode, preset, custom); err != nil {
		return err
	}

	var heroURL *string
	if snap.HeroImageURL != nil {
		t := strings.TrimSpace(*snap.HeroImageURL)
		if t != "" {
			heroURL = &t
		}
	}
	var heroPos *string
	if snap.HeroImageObjectPosition != nil {
		t := strings.TrimSpace(*snap.HeroImageObjectPosition)
		if t != "" {
			heroPos = &t
		}
	}
	if _, err := course.SetHeroImage(ctx, pool, courseCode, course.HeroImagePatch{
		ImageURL:             heroURL,
		ObjectPosition:       heroPos,
		UpdateImageURL:       true,
		UpdateObjectPosition: true,
	}); err != nil {
		return err
	}

	// Apply export feature flags; preserve flags not present in the v1 snapshot.
	if _, err := course.PatchFeatures(ctx, pool, courseCode,
		snap.NotebookEnabled,
		snap.FeedEnabled,
		snap.CalendarEnabled,
		snap.QuestionBankEnabled,
		snap.LockdownModeEnabled,
		snap.StandardsAlignmentEnabled,
		snap.AdaptivePathsEnabled,
		snap.SrsEnabled,
		snap.DiagnosticAssessmentsEnabled,
		snap.HintScaffoldingEnabled,
		snap.MisconceptionDetectionEnabled,
		cur.DiscussionsEnabled,
		cur.CollabDocsEnabled,
		cur.LiveSessionsEnabled,
		cur.GroupSpacesEnabled,
		cur.OfficeHoursEnabled,
		cur.AiTutorEnabled,
		cur.ModulesAiAssistantEnabled,
		cur.MultilingualMessagingEnabled,
		cur.FilesEnabled,
		cur.AttendanceEnabled,
		cur.WhiteboardEnabled,
		cur.ReportCardsEnabled,
		cur.VisualBoardsEnabled,
		cur.InteractiveQuizzesEnabled,
		cur.ScreenShareEnabled,
		snap.AdaptiveContentEnabled,
		snap.ContentToolsEnabled,
	); err != nil {
		return err
	}

	ct := strings.TrimSpace(snap.CourseType)
	if ct == "" {
		ct = "traditional"
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.courses SET course_type = $1, updated_at = NOW() WHERE course_code = $2
`, ct, courseCode); err != nil {
		return err
	}
	return nil
}

func mergeSyllabusSections(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode string,
	courseID uuid.UUID,
	incoming []coursesyllabus.SyllabusSection,
) error {
	payload, err := course.GetSyllabusByCourseCode(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	var current []course.SyllabusSection
	requireAcceptance := false
	if payload != nil {
		current = payload.Sections
		requireAcceptance = payload.RequireSyllabusAcceptance
	}
	seen := map[string]struct{}{}
	for _, s := range current {
		seen[s.ID] = struct{}{}
	}
	out := append([]course.SyllabusSection{}, current...)
	for _, s := range incoming {
		if _, ok := seen[s.ID]; ok {
			continue
		}
		seen[s.ID] = struct{}{}
		out = append(out, course.SyllabusSection{
			ID:       s.ID,
			Heading:  s.Heading,
			Markdown: s.Markdown,
		})
	}
	if len(out) > maxSyllabusSections {
		return InvalidInput(fmt.Sprintf("Too many syllabus sections after merge (max %d).", maxSyllabusSections))
	}
	if err := validateSyllabusSections(out); err != nil {
		return err
	}
	_, err = course.UpsertSyllabus(ctx, pool, courseID, out, &requireAcceptance)
	return err
}

// applyModuleBodies writes content for structure items. If onlyIDs is non-nil, only those item ids are applied.
func applyModuleBodies(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, ex *Bundle, onlyIDs map[uuid.UUID]struct{}) error {
	for _, it := range ex.Structure {
		itemID, err := uuid.Parse(it.ID)
		if err != nil {
			continue
		}
		if onlyIDs != nil {
			if _, ok := onlyIDs[itemID]; !ok {
				continue
			}
		}
		switch it.Kind {
		case "content_page":
			body, ok := ex.ContentPages[itemID]
			if !ok {
				continue
			}
			if err := coursemodulecontent.UpsertImportBody(ctx, pool, courseID, itemID, body.Markdown); err != nil {
				return err
			}
			if err := coursestructure.SetItemDueAt(ctx, pool, courseID, itemID, "content_page", body.DueAt); err != nil {
				return err
			}
		case "assignment":
			body, ok := ex.Assignments[itemID]
			if !ok {
				continue
			}
			var rubric []byte
			if len(body.Rubric) > 0 {
				rubric = append([]byte(nil), body.Rubric...)
			}
			if err := coursemoduleassignments.UpsertImportBody(ctx, pool, courseID, itemID,
				body.Markdown, body.PointsWorth, body.AvailableFrom, body.AvailableUntil,
				body.AssignmentAccessCode,
				body.SubmissionAllowText, body.SubmissionAllowFileUpload, body.SubmissionAllowURL,
				body.LateSubmissionPolicy, body.LatePenaltyPercent, rubric,
				body.OriginalityDetection, body.OriginalityStudentVisibility,
				body.BlindGrading, body.GradingType,
			); err != nil {
				return err
			}
			if err := coursestructure.SetItemDueAt(ctx, pool, courseID, itemID, "assignment", body.DueAt); err != nil {
				return err
			}
		case "quiz":
			body, ok := ex.Quizzes[itemID]
			if !ok {
				continue
			}
			if err := coursemodulequizzes.UpsertImportBody(ctx, pool, courseID, itemID, coursemodulequizzes.ImportQuizBody{
				Markdown:                    body.Markdown,
				Questions:                   body.Questions,
				AvailableFrom:               body.AvailableFrom,
				AvailableUntil:              body.AvailableUntil,
				UnlimitedAttempts:           body.UnlimitedAttempts,
				MaxAttempts:                 body.MaxAttempts,
				GradeAttemptPolicy:          body.GradeAttemptPolicy,
				PassingScorePercent:         body.PassingScorePercent,
				PointsWorth:                 body.PointsWorth,
				LateSubmissionPolicy:        body.LateSubmissionPolicy,
				LatePenaltyPercent:          body.LatePenaltyPercent,
				TimeLimitMinutes:            body.TimeLimitMinutes,
				TimerPauseWhenTabHidden:     body.TimerPauseWhenTabHidden,
				PerQuestionTimeLimitSeconds: body.PerQuestionTimeLimitSeconds,
				ShowScoreTiming:             body.ShowScoreTiming,
				ReviewVisibility:            body.ReviewVisibility,
				ReviewWhen:                  body.ReviewWhen,
				OneQuestionAtATime:          body.OneQuestionAtATime,
				ShuffleQuestions:            body.ShuffleQuestions,
				ShuffleChoices:              body.ShuffleChoices,
				AllowBackNavigation:         body.AllowBackNavigation,
				QuizAccessCode:              body.QuizAccessCode,
				AdaptiveDifficulty:          body.AdaptiveDifficulty,
				AdaptiveTopicBalance:        body.AdaptiveTopicBalance,
				AdaptiveStopRule:            body.AdaptiveStopRule,
				RandomQuestionPoolCount:     body.RandomQuestionPoolCount,
				LockdownMode:                body.LockdownMode,
				FocusLossThreshold:          body.FocusLossThreshold,
				IsAdaptive:                  body.IsAdaptive,
				AdaptiveSystemPrompt:        body.AdaptiveSystemPrompt,
				AdaptiveSourceItemIDs:       body.AdaptiveSourceItemIDs,
				AdaptiveQuestionCount:       body.AdaptiveQuestionCount,
				AdaptiveDeliveryMode:        body.AdaptiveDeliveryMode,
			}); err != nil {
				return err
			}
			if err := coursestructure.SetItemDueAt(ctx, pool, courseID, itemID, "quiz", body.DueAt); err != nil {
				return err
			}
		case "external_link":
			raw := ""
			if it.ExternalURL != nil {
				raw = strings.TrimSpace(*it.ExternalURL)
			}
			stored := ""
			if raw != "" {
				s, err := coursemoduleexternallinks.ValidateExternalHTTPURL(raw)
				if err != nil {
					return InvalidInput(err.Error())
				}
				stored = s
			}
			if err := coursemoduleexternallinks.UpsertImportBody(ctx, pool, courseID, itemID, stored); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyEnrollmentsFromExport(
	ctx context.Context,
	pool *pgxpool.Pool,
	targetCourseCode string,
	courseID uuid.UUID,
	mode courseexport.CourseImportMode,
	rows []courseexport.ExportedCourseEnrollment,
) error {
	if len(rows) == 0 {
		return nil
	}
	creator, err := enrollment.GetCourseCreatorUserID(ctx, pool, targetCourseCode)
	if err != nil {
		return err
	}
	if creator == nil {
		return InvalidInput("Course is missing a creator; cannot apply enrollments.")
	}
	ph, err := authservice.PlaceholderPasswordHash()
	if err != nil {
		return err
	}

	if mode == courseexport.CourseImportModeErase || mode == courseexport.CourseImportModeOverwrite {
		if err := enrollment.DeleteEnrollmentsExceptCreatorTeacher(ctx, pool, courseID, *creator); err != nil {
			return err
		}
	}
	for i := range rows {
		if err := applyOneEnrollmentFromExport(ctx, pool, targetCourseCode, courseID, *creator, &rows[i], ph); err != nil {
			return err
		}
	}
	return nil
}

func applyOneEnrollmentFromExport(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode string,
	courseID, creatorUserID uuid.UUID,
	row *courseexport.ExportedCourseEnrollment,
	placeholderPasswordHash string,
) error {
	email := strings.ToLower(strings.TrimSpace(row.Email))
	var displayName *string
	if row.DisplayName != nil {
		t := strings.TrimSpace(*row.DisplayName)
		if t != "" {
			displayName = &t
		}
	}

	u, createdUser, err := findOrCreateUserForImport(ctx, pool, email, displayName, placeholderPasswordHash)
	if err != nil {
		return err
	}
	if createdUser {
		_ = rbac.AssignUserRoleByName(ctx, pool, u, "Student")
	}

	role := strings.TrimSpace(row.Role)
	isCreator := u == creatorUserID

	if isCreator && (role == "student" || role == "instructor") {
		// Match roster API: do not add secondary student/instructor rows for the course creator.
		return nil
	}
	// Skip creator-only teacher re-apply path handled below via ensure.

	switch role {
	case "student":
		if err := enrollment.InsertStudentIfMissing(ctx, pool, courseID, u); err != nil {
			return err
		}
	case "instructor":
		// Map Teacher/TA grant to modern enrollment roles when useful.
		grantAs := "instructor"
		if row.InstructorGrantRole != nil {
			g := strings.TrimSpace(*row.InstructorGrantRole)
			if g == "TA" {
				grantAs = "ta"
			}
		}
		if grantAs == "ta" {
			if err := enrollment.InsertEnrollmentRoleIfMissing(ctx, pool, courseID, u, "ta"); err != nil {
				return err
			}
		} else {
			if err := enrollment.UpsertInstructorEnrollment(ctx, pool, courseID, u); err != nil {
				return err
			}
		}
		if err := refreshEnrollmentGrants(ctx, pool, courseCode, courseID, u); err != nil {
			return err
		}
	case "teacher", "owner":
		if err := enrollment.EnsureTeacherEnrollment(ctx, pool, courseID, u); err != nil {
			return err
		}
		if err := refreshEnrollmentGrants(ctx, pool, courseCode, courseID, u); err != nil {
			return err
		}
	case "ta", "designer", "observer", "auditor", "librarian":
		if err := enrollment.InsertEnrollmentRoleIfMissing(ctx, pool, courseID, u, role); err != nil {
			return err
		}
		if err := refreshEnrollmentGrants(ctx, pool, courseCode, courseID, u); err != nil {
			return err
		}
	}
	return nil
}

func findOrCreateUserForImport(ctx context.Context, pool *pgxpool.Pool, email string, displayName *string, passwordHash string) (uuid.UUID, bool, error) {
	existing, err := user.FindByEmailCI(ctx, pool, email)
	if err != nil {
		return uuid.Nil, false, err
	}
	if existing != nil {
		id, err := uuid.Parse(existing.ID)
		return id, false, err
	}
	row, err := user.InsertUser(ctx, pool, email, passwordHash, displayName)
	if err != nil {
		// Race: another import created the user.
		if existing2, fe := user.FindByEmailCI(ctx, pool, email); fe == nil && existing2 != nil {
			id, perr := uuid.Parse(existing2.ID)
			return id, false, perr
		}
		return uuid.Nil, false, err
	}
	id, err := uuid.Parse(row.ID)
	return id, true, err
}

func refreshEnrollmentGrants(ctx context.Context, pool *pgxpool.Pool, courseCode string, courseID, userID uuid.UUID) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, userID, courseID, courseCode); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
