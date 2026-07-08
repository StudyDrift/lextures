package introcourse

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notificationevents"
	credrepo "github.com/lextures/lextures/server/internal/repos/credentials"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	stprog "github.com/lextures/lextures/server/internal/repos/studentprogress"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	credsvc "github.com/lextures/lextures/server/internal/service/credentials"
	"github.com/lextures/lextures/server/internal/service/gamification"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

const (
	// CapstoneSlug is the curriculum slug for the required capstone reflection (IC05 completion rule).
	CapstoneSlug = "m7.finish.capstone"
	// DefaultCompletionGradeThreshold is the minimum running grade percent to complete (0 = auto-credit friendly).
	DefaultCompletionGradeThreshold = 0.0
)

// NextItem is the suggested deep link for continuing the intro course.
type NextItem struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Route string `json:"route"`
}

// ModuleProgress is one module row for the in-course progress rail (IC06).
type ModuleProgress struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"` // done | current | upcoming
}

// Progress is per-student intro course progress and completion state (IC05).
type Progress struct {
	Enrolled               bool             `json:"enrolled"`
	CourseCode             string           `json:"courseCode,omitempty"`
	ModulesComplete        int              `json:"modulesComplete"`
	ModulesTotal           int              `json:"modulesTotal"`
	Percent                int              `json:"percent"`
	RunningGrade           *float64         `json:"runningGrade,omitempty"`
	CompletedAt            *time.Time       `json:"completedAt,omitempty"`
	CredentialID           *uuid.UUID       `json:"-"`
	CredentialIDStr        *string          `json:"credentialId,omitempty"`
	NextItem               *NextItem        `json:"nextItem,omitempty"`
	Modules                []ModuleProgress `json:"modules,omitempty"`
	WelcomeBannerDismissed bool             `json:"welcomeBannerDismissed"`
	CelebrationSeen        bool             `json:"celebrationSeen"`
	JustCompleted          bool             `json:"-"`
}

// ModuleFunnelEntry is one row in admin analytics.
type ModuleFunnelEntry struct {
	ModuleSlug    string  `json:"moduleSlug"`
	ModuleTitle   string  `json:"moduleTitle"`
	QuizAttempted int     `json:"quizAttempted"`
	AttemptRate   float64 `json:"attemptRate"`
}

// Analytics is the admin aggregate for intro course completion (IC05/IC08).
type Analytics struct {
	Enrolled                 int                 `json:"enrolled"`
	Completed                int                 `json:"completed"`
	CompletionRate           float64             `json:"completionRate"`
	PerModuleFunnel          []ModuleFunnelEntry `json:"perModuleFunnel"`
	DropOffModuleSlug        string              `json:"dropOffModuleSlug,omitempty"`
	AvgTimeToCompleteHours   *float64            `json:"avgTimeToCompleteHours,omitempty"`
}

type moduleQuiz struct {
	ModuleSlug  string
	ModuleTitle string
	SortOrder   int
	QuizID      uuid.UUID
	QuizSlug    string
	QuizTitle   string
}

// LoadProgress returns derived progress for a learner in the intro course.
func LoadProgress(ctx context.Context, exec Execer, cfg config.Config, courseID, userID uuid.UUID) (Progress, error) {
	recordProgressRecompute()
	out := Progress{ModulesTotal: 7}
	if exec == nil || !cfg.IntroCourseEnabled || courseID == uuid.Nil || userID == uuid.Nil {
		return out, nil
	}

	enrolled, err := isEnrolledStudent(ctx, exec, courseID, userID)
	if err != nil {
		return out, err
	}
	out.Enrolled = enrolled
	if !enrolled {
		return out, nil
	}
	out.CourseCode = CourseCode

	ui, err := icrepo.GetUIState(ctx, exec, userID)
	if err != nil {
		return out, err
	}
	out.WelcomeBannerDismissed = ui.WelcomeBannerDismissed
	out.CelebrationSeen = ui.CelebrationSeen

	quizzes, err := listModuleQuizzes(ctx, exec, courseID)
	if err != nil {
		return out, err
	}
	out.ModulesTotal = len(quizzes)
	if out.ModulesTotal == 0 {
		out.ModulesTotal = 7
	}

	for _, qz := range quizzes {
		done, err := quizAttempted(ctx, exec, courseID, qz.QuizID, userID)
		if err != nil {
			return out, err
		}
		if done {
			out.ModulesComplete++
		}
	}
	if out.ModulesTotal > 0 {
		out.Percent = (out.ModulesComplete * 100) / out.ModulesTotal
	}

	if pool, ok := exec.(*pgxpool.Pool); ok && pool != nil {
		avg, err := stprog.AvgGradePercent(ctx, pool, courseID, userID)
		if err != nil {
			return out, err
		}
		out.RunningGrade = avg
	}

	completion, err := icrepo.GetCompletion(ctx, exec, userID)
	if err != nil {
		return out, err
	}
	if completion != nil {
		t := completion.CompletedAt.UTC()
		out.CompletedAt = &t
		if completion.CredentialID != nil {
			id := completion.CredentialID.String()
			out.CredentialIDStr = &id
		}
	}

	modules, err := buildModuleProgress(ctx, exec, courseID, userID, quizzes)
	if err != nil {
		return out, err
	}
	out.Modules = modules

	if out.CompletedAt == nil {
		next, err := firstIncompleteItem(ctx, exec, cfg, courseID, userID, quizzes)
		if err != nil {
			return out, err
		}
		out.NextItem = next
	}
	return out, nil
}

func buildModuleProgress(ctx context.Context, exec Execer, courseID, userID uuid.UUID, quizzes []moduleQuiz) ([]ModuleProgress, error) {
	if len(quizzes) == 0 {
		return nil, nil
	}
	foundCurrent := false
	out := make([]ModuleProgress, 0, len(quizzes))
	for _, qz := range quizzes {
		done, err := quizAttempted(ctx, exec, courseID, qz.QuizID, userID)
		if err != nil {
			return nil, err
		}
		status := "upcoming"
		switch {
		case done:
			status = "done"
		case !foundCurrent:
			status = "current"
			foundCurrent = true
		}
		out = append(out, ModuleProgress{
			Slug:   qz.ModuleSlug,
			Title:  qz.ModuleTitle,
			Status: status,
		})
	}
	return out, nil
}

// RecheckCompletion evaluates the completion rule and records completion, credential, and event once.
func RecheckCompletion(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, userID uuid.UUID) (Progress, error) {
	if pool == nil || !cfg.IntroCourseEnabled || courseID == uuid.Nil || userID == uuid.Nil {
		return Progress{}, nil
	}
	prog, err := LoadProgress(ctx, pool, cfg, courseID, userID)
	if err != nil {
		return prog, err
	}
	if prog.CompletedAt != nil {
		return prog, nil
	}
	if !prog.Enrolled {
		return prog, nil
	}

	meets, err := meetsCompletionRule(ctx, pool, cfg, courseID, userID, prog.RunningGrade)
	if err != nil || !meets {
		return prog, err
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return prog, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	justInserted, row, err := icrepo.InsertCompletionIfAbsent(ctx, tx, userID, prog.RunningGrade)
	if err != nil {
		return prog, err
	}
	if !justInserted {
		if err := tx.Commit(ctx); err != nil {
			return prog, err
		}
		return LoadProgress(ctx, pool, cfg, courseID, userID)
	}

	if err := tx.Commit(ctx); err != nil {
		return prog, err
	}

	recordCompletion()
	setCompletionRateGauge(ctx, pool, courseID)

	var credID *uuid.UUID
	if cfg.FFCompletionCredentials {
		name, _ := learnerDisplayName(ctx, pool, userID)
		cred, issueErr := credsvc.IssueCourseCompletion(ctx, pool, cfg, credsvc.IssueCourseParams{
			RecipientID: userID,
			LearnerName: name,
			CourseID:    courseID,
		})
		if issueErr != nil {
			slog.Warn("intro course credential issue failed", "user_id", userID, "err", issueErr)
		} else if cred != nil {
			recordCredentialIssued()
			credID = &cred.ID
			_ = icrepo.SetCredentialID(ctx, pool, userID, cred.ID)
		}
	}

	if row != nil && !row.EventSent {
		if err := emitCompletionEvent(ctx, cfg, pool, courseID, userID, credID); err != nil {
			slog.Warn("intro course completion event failed", "user_id", userID, "err", err)
		} else {
			_ = icrepo.MarkEventSent(ctx, pool, userID)
		}
	}

	prog, err = LoadProgress(ctx, pool, cfg, courseID, userID)
	if err != nil {
		return prog, err
	}
	prog.JustCompleted = true
	return prog, nil
}

// LoadAnalytics returns admin completion-rate and per-module funnel data.
func LoadAnalytics(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (Analytics, error) {
	var out Analytics
	if pool == nil || courseID == uuid.Nil {
		return out, nil
	}
	enrolled, err := icrepo.CountEnrolledStudents(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	completed, err := icrepo.CountCompleted(ctx, pool)
	if err != nil {
		return out, err
	}
	out.Enrolled = enrolled
	out.Completed = completed
	if enrolled > 0 {
		out.CompletionRate = float64(completed) / float64(enrolled)
	}

	rows, err := icrepo.ListModuleFunnel(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	var prevRate float64 = 1
	var maxDrop float64
	for _, r := range rows {
		rate := 0.0
		if enrolled > 0 {
			rate = float64(r.QuizAttempted) / float64(enrolled)
		}
		out.PerModuleFunnel = append(out.PerModuleFunnel, ModuleFunnelEntry{
			ModuleSlug:    r.ModuleSlug,
			ModuleTitle:   r.ModuleTitle,
			QuizAttempted: r.QuizAttempted,
			AttemptRate:   rate,
		})
		if drop := prevRate - rate; drop > maxDrop {
			maxDrop = drop
			out.DropOffModuleSlug = r.ModuleSlug
		}
		prevRate = rate
	}
	avgHours, err := icrepo.AvgCompletionHours(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	out.AvgTimeToCompleteHours = avgHours
	return out, nil
}

// SweepIncompleteCompletions re-checks completion for all enrolled students (nightly job).
func SweepIncompleteCompletions(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID uuid.UUID) (int, error) {
	if pool == nil || !cfg.IntroCourseEnabled || courseID == uuid.Nil {
		return 0, nil
	}
	ids, err := icrepo.ListEnrolledStudentIDs(ctx, pool, courseID)
	if err != nil {
		return 0, err
	}
	var n int
	for _, uid := range ids {
		done, err := icrepo.HasCompleted(ctx, pool, uid)
		if err != nil {
			return n, err
		}
		if done {
			continue
		}
		prog, err := RecheckCompletion(ctx, pool, cfg, courseID, uid)
		if err != nil {
			slog.Warn("intro course completion sweep error", "user_id", uid, "err", err)
			continue
		}
		if prog.JustCompleted {
			n++
		}
	}
	return n, nil
}

// ShouldNudgeIntroCourse reports whether onboarding nudges should target this user (IC05 AC-5).
func ShouldNudgeIntroCourse(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID) (bool, error) {
	if pool == nil || !cfg.IntroCourseEnabled || userID == uuid.Nil {
		return false, nil
	}
	done, err := icrepo.HasCompleted(ctx, pool, userID)
	if err != nil {
		return false, err
	}
	return !done, nil
}

func meetsCompletionRule(ctx context.Context, exec Execer, cfg config.Config, courseID, userID uuid.UUID, runningGrade *float64) (bool, error) {
	quizzes, err := listModuleQuizzes(ctx, exec, courseID)
	if err != nil {
		return false, err
	}
	for _, qz := range quizzes {
		done, err := quizAttempted(ctx, exec, courseID, qz.QuizID, userID)
		if err != nil {
			return false, err
		}
		if !done {
			return false, nil
		}
	}

	capstoneID, err := lookupItemIDBySlug(ctx, exec, CapstoneSlug)
	if err != nil || capstoneID == nil {
		return false, err
	}
	submitted, err := assignmentSubmitted(ctx, exec, courseID, *capstoneID, userID)
	if err != nil {
		return false, err
	}
	if !submitted {
		return false, nil
	}

	threshold := DefaultCompletionGradeThreshold
	if runningGrade == nil {
		if threshold <= 0 {
			return true, nil
		}
		return false, nil
	}
	return *runningGrade >= threshold, nil
}

func isEnrolledStudent(ctx context.Context, exec Execer, courseID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := exec.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.course_enrollments
    WHERE course_id = $1 AND user_id = $2 AND role = 'student' AND active AND state = 'active'
)
`, courseID, userID).Scan(&ok)
	return ok, err
}

func listModuleQuizzes(ctx context.Context, exec Execer, courseID uuid.UUID) ([]moduleQuiz, error) {
	rows, err := exec.Query(ctx, `
SELECT mod_ici.slug, mod_csi.title, mod_csi.sort_order,
       quiz_csi.id, quiz_ici.slug, quiz_csi.title
FROM settings.intro_course_items quiz_ici
INNER JOIN course.course_structure_items quiz_csi ON quiz_csi.id = quiz_ici.structure_item_id
INNER JOIN course.course_structure_items mod_csi ON mod_csi.id = quiz_csi.parent_id
INNER JOIN settings.intro_course_items mod_ici ON mod_ici.structure_item_id = mod_csi.id
WHERE quiz_csi.course_id = $1
  AND quiz_csi.kind = 'quiz' AND quiz_csi.published AND NOT quiz_csi.archived
  AND mod_csi.kind = 'module' AND mod_csi.published AND NOT mod_csi.archived
  AND quiz_ici.slug LIKE '%.knowledge-check'
ORDER BY mod_csi.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []moduleQuiz
	for rows.Next() {
		var q moduleQuiz
		if err := rows.Scan(&q.ModuleSlug, &q.ModuleTitle, &q.SortOrder, &q.QuizID, &q.QuizSlug, &q.QuizTitle); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func quizAttempted(ctx context.Context, exec Execer, courseID, itemID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := exec.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.quiz_attempts
    WHERE course_id = $1 AND structure_item_id = $2 AND student_user_id = $3 AND status = 'submitted'
)
`, courseID, itemID, userID).Scan(&ok)
	return ok, err
}

func assignmentSubmitted(ctx context.Context, exec Execer, courseID, itemID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := exec.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.module_assignment_submissions
    WHERE course_id = $1 AND module_item_id = $2 AND submitted_by = $3
)
`, courseID, itemID, userID).Scan(&ok)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	err = exec.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.course_grades
    WHERE course_id = $1 AND module_item_id = $2 AND student_user_id = $3
)
`, courseID, itemID, userID).Scan(&ok)
	return ok, err
}

func lookupItemIDBySlug(ctx context.Context, exec Execer, slug string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := exec.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = $1
`, slug).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func firstIncompleteItem(ctx context.Context, exec Execer, cfg config.Config, courseID, userID uuid.UUID, quizzes []moduleQuiz) (*NextItem, error) {
	for _, qz := range quizzes {
		done, err := quizAttempted(ctx, exec, courseID, qz.QuizID, userID)
		if err != nil {
			return nil, err
		}
		if done {
			continue
		}
		page, err := firstModulePage(ctx, exec, courseID, qz.ModuleSlug)
		if err != nil {
			return nil, err
		}
		if page != nil {
			return page, nil
		}
		return itemRoute(qz.QuizSlug, qz.QuizTitle, qz.QuizID, "quiz"), nil
	}

	capstoneID, err := lookupItemIDBySlug(ctx, exec, CapstoneSlug)
	if err != nil || capstoneID == nil {
		return nil, err
	}
	submitted, err := assignmentSubmitted(ctx, exec, courseID, *capstoneID, userID)
	if err != nil || submitted {
		return nil, err
	}
	var title, slug string
	err = exec.QueryRow(ctx, `
SELECT ici.slug, csi.title
FROM settings.intro_course_items ici
INNER JOIN course.course_structure_items csi ON csi.id = ici.structure_item_id
WHERE ici.slug = $1
`, CapstoneSlug).Scan(&slug, &title)
	if err != nil {
		return nil, err
	}
	return itemRoute(slug, title, *capstoneID, "assignment"), nil
}

func firstModulePage(ctx context.Context, exec Execer, courseID uuid.UUID, moduleSlug string) (*NextItem, error) {
	var itemID uuid.UUID
	var slug, title string
	err := exec.QueryRow(ctx, `
SELECT page_csi.id, page_ici.slug, page_csi.title
FROM settings.intro_course_items mod_ici
INNER JOIN course.course_structure_items mod_csi ON mod_csi.id = mod_ici.structure_item_id
INNER JOIN course.course_structure_items page_csi ON page_csi.parent_id = mod_csi.id
INNER JOIN settings.intro_course_items page_ici ON page_ici.structure_item_id = page_csi.id
WHERE mod_ici.slug = $1 AND mod_csi.course_id = $2
  AND page_csi.kind = 'content_page' AND page_csi.published AND NOT page_csi.archived
ORDER BY page_csi.sort_order
LIMIT 1
`, moduleSlug, courseID).Scan(&itemID, &slug, &title)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return itemRoute(slug, title, itemID, "content_page"), nil
}

func itemRoute(slug, title string, itemID uuid.UUID, kind string) *NextItem {
	segment := "content"
	switch kind {
	case "quiz":
		segment = "quiz"
	case "assignment":
		segment = "assignment"
	case "content_page":
		segment = "content"
	}
	return &NextItem{
		Slug:  slug,
		Title: title,
		Route: fmt.Sprintf("/courses/%s/modules/%s/%s", CourseCode, segment, itemID.String()),
	}
}

func emitCompletionEvent(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, courseID, userID uuid.UUID, credID *uuid.UUID) error {
	gamification.EmitCourseCompleted(pool, cfg, userID, courseID)

	title := "You've completed Welcome to Lextures"
	body := "Congratulations — you've finished the onboarding course."
	actionURL := fmt.Sprintf("/courses/%s", CourseCode)
	if credID != nil {
		actionURL = "/me/credentials"
		body = "Congratulations — your onboarding certificate is ready."
	}
	_, _ = notificationsinbox.Insert(ctx, pool, userID, notificationevents.IntroCourseCompleted,
		title, body, actionURL)

	if cfg.EmailNotificationsEnabled && pool != nil {
		vars := map[string]string{
			"courseName": Title,
			"courseUrl":  strings.TrimRight(cfg.PublicWebOrigin, "/") + "/courses/" + CourseCode,
		}
		if credID != nil {
			cred, err := credrepo.GetByID(ctx, pool, *credID)
			if err == nil && cred != nil {
				vars["credentialName"] = cred.Title
				vars["credentialsUrl"] = strings.TrimRight(cfg.PublicWebOrigin, "/") + "/me/credentials"
				vars["verifyUrl"] = credsvc.VerificationURL(cfg.PublicWebOrigin, cred.ID)
			}
		}
		ns := notifications.Service{Pool: pool, Config: cfg}
		_ = ns.EnqueueEmail(ctx, userID, notificationevents.IntroCourseCompleted, "intro_course_completed", vars, nil)
	}
	return nil
}

func learnerDisplayName(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	row, err := userrepo.FindByID(ctx, pool, userID)
	if err != nil || row == nil {
		return "", err
	}
	if row.DisplayName != nil && strings.TrimSpace(*row.DisplayName) != "" {
		return strings.TrimSpace(*row.DisplayName), nil
	}
	return row.Email, nil
}

func setCompletionRateGauge(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) {
	enrolled, err := icrepo.CountEnrolledStudents(ctx, pool, courseID)
	if err != nil || enrolled == 0 {
		return
	}
	completed, err := icrepo.CountCompleted(ctx, pool)
	if err != nil {
		return
	}
	setCompletionRate(float64(completed) / float64(enrolled))
}