// Package learningpaths implements learning path enrollment and progress (plan 15.4).
package learningpaths

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	repoCCR "github.com/lextures/lextures/server/internal/repos/ccr"
	repoPaths "github.com/lextures/lextures/server/internal/repos/learningpaths"
	"github.com/lextures/lextures/server/internal/config"
	credsvc "github.com/lextures/lextures/server/internal/service/credentials"
	"github.com/lextures/lextures/server/internal/service/gamification"
	"github.com/lextures/lextures/server/internal/courseroles"
)

// ProgressOptions carries optional credential issuance context (plan 15.5).
type ProgressOptions struct {
	Cfg         config.Config
	LearnerName string
}

var (
	ErrPathNotFound       = errors.New("learning path not found")
	ErrAlreadyEnrolled    = errors.New("already enrolled in path")
	ErrEntitlementRequired = errors.New("path bundle purchase required")
	ErrCourseNotOwned     = errors.New("course not owned by creator")
)

// CourseProgress is per-course completion within a path.
type CourseProgress struct {
	CourseID    uuid.UUID
	Position    int
	CourseCode  string
	Title       string
	Completed   bool
	Recommended bool
}

// PathProgress is aggregate learner progress for a path.
type PathProgress struct {
	PathID           uuid.UUID
	PathTitle        string
	Slug             string
	TotalCourses     int
	CompletedCourses int
	Percent          int
	CompletedAt      *time.Time
	JustCompleted    bool
	Courses          []CourseProgress
}

// EnrollUser enrolls a learner in a path and all constituent courses atomically.
func EnrollUser(ctx context.Context, pool *pgxpool.Pool, pathID, userID uuid.UUID) (*repoPaths.PathEnrollment, error) {
	p, err := repoPaths.GetPathByID(ctx, pool, pathID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPathNotFound
	}
	existing, err := repoPaths.GetPathEnrollment(ctx, pool, userID, pathID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyEnrolled
	}
	if repoPaths.PathRequiresPayment(p) {
		ok, err := repoPaths.HasPathEntitlement(ctx, pool, userID, pathID)
		if err != nil {
			return nil, err
		}
		if !ok && p.CreatorID != userID {
			return nil, ErrEntitlementRequired
		}
	}
	courses, err := repoPaths.ListPathCourses(ctx, pool, pathID)
	if err != nil {
		return nil, err
	}
	if len(courses) == 0 {
		return nil, fmt.Errorf("path has no courses")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var enrollment repoPaths.PathEnrollment
	err = tx.QueryRow(ctx, `
INSERT INTO learningpath.path_enrollments (user_id, path_id)
VALUES ($1, $2)
RETURNING id, user_id, path_id, enrolled_at, completed_at
`, userID, pathID).Scan(
		&enrollment.ID, &enrollment.UserID, &enrollment.PathID, &enrollment.EnrolledAt, &enrollment.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	for _, c := range courses {
		var exists bool
		if err := tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.course_enrollments ce
  WHERE ce.course_id = $1 AND ce.user_id = $2 AND ce.active
)
`, c.CourseID, userID).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		tag, err := tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active)
VALUES ($1, $2, 'student', true)
`, c.CourseID, userID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, fmt.Errorf("course enrollment not created")
		}
		if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, userID, c.CourseID, c.CourseCode); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	RecordEnrollment()
	return &enrollment, nil
}

// GetProgress returns path-level progress for a learner.
func GetProgress(ctx context.Context, pool *pgxpool.Pool, userID, pathID uuid.UUID, opts *ProgressOptions) (*PathProgress, error) {
	p, err := repoPaths.GetPathByID(ctx, pool, pathID)
	if err != nil || p == nil {
		return nil, err
	}
	enrollment, err := repoPaths.GetPathEnrollment(ctx, pool, userID, pathID)
	if err != nil {
		return nil, err
	}
	courses, err := repoPaths.ListPathCourses(ctx, pool, pathID)
	if err != nil {
		return nil, err
	}
	completedSet, err := completedCourseSet(ctx, pool, userID)
	if err != nil {
		return nil, err
	}

	prog := &PathProgress{
		PathID:    pathID,
		PathTitle: p.Title,
		TotalCourses: len(courses),
	}
	if p.Slug != nil {
		prog.Slug = *p.Slug
	}
	if enrollment != nil {
		prog.CompletedAt = enrollment.CompletedAt
	}

	firstIncomplete := -1
	for _, c := range courses {
		done := completedSet[c.CourseID]
		if done {
			prog.CompletedCourses++
		} else if firstIncomplete < 0 {
			firstIncomplete = c.Position
		}
		cp := CourseProgress{
			CourseID:   c.CourseID,
			Position:   c.Position,
			CourseCode: c.CourseCode,
			Title:      c.Title,
			Completed:  done,
		}
		if firstIncomplete < 0 || c.Position <= firstIncomplete {
			cp.Recommended = true
		}
		prog.Courses = append(prog.Courses, cp)
	}
	if prog.TotalCourses > 0 {
		prog.Percent = (prog.CompletedCourses * 100) / prog.TotalCourses
	}

	if enrollment != nil && enrollment.CompletedAt == nil && prog.CompletedCourses == prog.TotalCourses && prog.TotalCourses > 0 {
		justDone, err := repoPaths.MarkPathCompleted(ctx, pool, enrollment.ID)
		if err != nil {
			return nil, err
		}
		if justDone {
			now := time.Now().UTC()
			prog.CompletedAt = &now
			prog.JustCompleted = true
			RecordCompletion()
			if opts != nil {
				gamification.EmitPathCompleted(pool, opts.Cfg, userID, pathID)
			}
			if err := issuePathCertificate(ctx, pool, userID, p, opts); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
	}
	return prog, nil
}

func completedCourseSet(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	completions, err := repoCCR.ListCourseCompletions(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]bool, len(completions))
	for _, c := range completions {
		out[c.CourseID] = true
	}
	return out, nil
}

func issuePathCertificate(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, p *repoPaths.Path, opts *ProgressOptions) error {
	if opts != nil && opts.Cfg.FFCompletionCredentials {
		_, err := credsvc.IssuePathCompletion(ctx, pool, opts.Cfg, credsvc.IssuePathParams{
			RecipientID: userID,
			LearnerName: opts.LearnerName,
			PathID:      p.ID,
			PathTitle:   p.Title,
		})
		return err
	}
	sourceID := p.ID
	title := p.Title + " — Learning Path"
	_, err := repoCCR.CreateAchievement(ctx, pool, repoCCR.Achievement{
		UserID:          userID,
		AchievementType: repoCCR.TypeCertificate,
		SourceID:        &sourceID,
		Title:           title,
		IssuedAt:        time.Now().UTC(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// ValidateCreatorCourses ensures the creator teaches every course in the list.
func ValidateCreatorCourses(ctx context.Context, pool *pgxpool.Pool, creatorID uuid.UUID, courseIDs []uuid.UUID) error {
	for _, cid := range courseIDs {
		ok, err := repoPaths.UserTeachesCourse(ctx, pool, creatorID, cid)
		if err != nil {
			return err
		}
		if !ok {
			return ErrCourseNotOwned
		}
	}
	return nil
}

// CalcProgressPercent returns completed/total as an integer percent.
func CalcProgressPercent(completed, total int) int {
	if total <= 0 {
		return 0
	}
	return (completed * 100) / total
}
