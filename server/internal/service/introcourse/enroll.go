package introcourse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// Execer is satisfied by *pgxpool.Pool and pgx.Tx so callers choose atomicity.
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// EnrollPath identifies the user-creation source for enrollment metrics (IC02).
type EnrollPath string

const (
	PathSignup      EnrollPath = "signup"
	PathSSO         EnrollPath = "sso"
	PathClever      EnrollPath = "clever"
	PathCanvas      EnrollPath = "canvas"
	PathAdminImport EnrollPath = "admin_import"
	PathBackfill    EnrollPath = "backfill"
)

const backfillBatchSize = 500

// SkipParentsOnEnroll controls whether parent accounts are excluded from intro enrollment (IC02 FR-4).
var SkipParentsOnEnroll = true

// EnsureEnrollment enrolls userID as a student when the flag is on and the user is eligible.
// Idempotent via ON CONFLICT on (course_id, user_id, role).
//
// Call sites (keep in sync when adding user-creation paths):
//   - authservice.Signup (credentials.go)
//   - browsersaml ACS provisioning (acs.go)
//   - oidcauth OIDC and K12 login provisioning (oidcauth.go, k12_login.go)
//   - cleverauth Clever provisioning (cleverauth.go)
//   - canvas enrollment import account creation (canvas_enrollment_import.go)
//   - csvimport bulk user create (csvimport/service.go)
//   - provisioning/scim CreateUser (users.go)
//   - platformpeople admin invite (platform_people.go)
func (s *Service) EnsureEnrollment(ctx context.Context, cfg config.Config, exec Execer, userID uuid.UUID, path EnrollPath) error {
	_, err := s.ensureEnrollment(ctx, cfg, exec, userID, path)
	return err
}

func (s *Service) ensureEnrollment(ctx context.Context, cfg config.Config, exec Execer, userID uuid.UUID, path EnrollPath) (bool, error) {
	if s == nil || s.Pool == nil {
		return false, nil
	}
	if !Enabled(cfg) {
		recordEnroll(string(path), "skipped_disabled")
		return false, nil
	}
	if userID == uuid.Nil || userID == SystemUserID {
		recordEnroll(string(path), "skipped_ineligible")
		return false, nil
	}

	courseID, ok, err := s.CourseID(ctx)
	if err != nil {
		recordEnroll(string(path), "error")
		return false, err
	}
	if !ok || courseID == uuid.Nil {
		if _, provErr := s.EnsureProvisioned(ctx, cfg); provErr != nil {
			recordEnroll(string(path), "error")
			return false, provErr
		}
		courseID, ok, err = s.CourseID(ctx)
		if err != nil || !ok || courseID == uuid.Nil {
			recordEnroll(string(path), "skipped_no_course")
			return false, err
		}
	}

	eligible, skipReason, err := checkEligibility(ctx, exec, userID, courseID)
	if err != nil {
		recordEnroll(string(path), "error")
		return false, err
	}
	if !eligible {
		recordEnroll(string(path), "skipped_"+skipReason)
		return false, nil
	}

	run := func(tx pgx.Tx) (bool, error) {
		created, err := enrollStudentTx(ctx, tx, courseID, userID, CourseCode)
		if err != nil {
			recordEnroll(string(path), "error")
			return false, err
		}
		if created {
			recordEnroll(string(path), "enrolled")
		} else {
			recordEnroll(string(path), "already_enrolled")
		}
		return created, nil
	}

	if tx, ok := exec.(pgx.Tx); ok {
		return run(tx)
	}

	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		recordEnroll(string(path), "error")
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	created, err := run(tx)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		recordEnroll(string(path), "error")
		return false, err
	}
	return created, nil
}

// EnsureEnrollmentBestEffort runs EnsureEnrollment without failing account creation (IC02 FR-3).
// On failure it logs, records metrics, and enqueues a retry job.
func EnsureEnrollmentBestEffort(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, exec Execer, userID uuid.UUID, path EnrollPath) {
	if pool == nil || userID == uuid.Nil {
		return
	}
	svc := New(pool)
	if err := svc.EnsureEnrollment(ctx, cfg, exec, userID, path); err != nil {
		slog.Warn("intro course enrollment failed",
			"user_id", userID,
			"path", path,
			"err", err,
		)
		if _, enqErr := EnqueueEnrollmentRetry(ctx, pool, userID, path); enqErr != nil {
			slog.Warn("intro course enrollment retry enqueue failed",
				"user_id", userID,
				"path", path,
				"err", enqErr,
			)
		}
	}
}

func checkEligibility(ctx context.Context, exec Execer, userID, courseID uuid.UUID) (bool, string, error) {
	var accountType string
	var orgID uuid.UUID
	var isInstructor bool
	err := exec.QueryRow(ctx, `
SELECT u.account_type, u.org_id,
       EXISTS (
           SELECT 1 FROM course.course_enrollments ce
           WHERE ce.course_id = $2 AND ce.user_id = $1
             AND ce.role IN ('teacher', 'owner', 'instructor')
       )
FROM "user".users u
WHERE u.id = $1
`, userID, courseID).Scan(&accountType, &orgID, &isInstructor)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "missing_user", nil
	}
	if err != nil {
		return false, "", err
	}
	if accountType == user.AccountTypeSystem {
		return false, "system", nil
	}
	if SkipParentsOnEnroll && accountType == user.AccountTypeParent {
		return false, "parent", nil
	}
	if isInstructor {
		return false, "instructor", nil
	}
	if orgID != organization.SeedDefaultOrgID {
		return false, "other_org", nil
	}
	return true, "", nil
}

func enrollStudentTx(ctx context.Context, tx pgx.Tx, courseID, userID uuid.UUID, courseCode string) (bool, error) {
	tag, err := tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'student')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, userID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 {
		if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, userID, courseID, courseCode); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// BackfillStatus is the admin-visible backfill progress snapshot.
type BackfillStatus struct {
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	EnrolledCount int64      `json:"enrolledCount"`
	Remaining     int64      `json:"remaining"`
}

// BackfillStatus returns persisted backfill progress (IC02 FR-7).
func (s *Service) BackfillStatus(ctx context.Context, cfg config.Config) (BackfillStatus, error) {
	if s == nil || s.Pool == nil {
		return BackfillStatus{}, nil
	}
	st, err := icrepo.LoadBackfillState(ctx, s.Pool)
	if err != nil {
		return BackfillStatus{}, err
	}
	out := BackfillStatus{
		StartedAt:     st.StartedAt,
		CompletedAt:   st.CompletedAt,
		EnrolledCount: st.EnrolledCount,
	}
	if !Enabled(cfg) {
		return out, nil
	}
	courseID, ok, err := s.CourseID(ctx)
	if err != nil || !ok {
		return out, err
	}
	remaining, err := icrepo.CountBackfillRemaining(ctx, s.Pool, courseID, organization.SeedDefaultOrgID, SkipParentsOnEnroll)
	if err != nil {
		return out, err
	}
	out.Remaining = remaining
	setBackfillRemaining(float64(remaining))
	if st.LastUserID != nil {
		setBackfillProgress(1)
	}
	return out, nil
}

// RunBackfill enrolls all eligible existing users in batches (IC02 FR-5).
func (s *Service) RunBackfill(ctx context.Context, cfg config.Config) error {
	if s == nil || s.Pool == nil {
		return nil
	}
	if !Enabled(cfg) {
		return nil
	}
	st, err := icrepo.LoadBackfillState(ctx, s.Pool)
	if err != nil {
		return err
	}
	if st.CompletedAt != nil {
		slog.Debug("intro course backfill already completed")
		return nil
	}

	courseID, ok, err := s.CourseID(ctx)
	if err != nil {
		return err
	}
	if !ok || courseID == uuid.Nil {
		if _, err := s.EnsureProvisioned(ctx, cfg); err != nil {
			return err
		}
		courseID, ok, err = s.CourseID(ctx)
		if err != nil || !ok {
			return fmt.Errorf("intro course backfill: course unavailable: %w", err)
		}
	}

	now := time.Now().UTC()
	if err := icrepo.EnsureBackfillStarted(ctx, s.Pool, now); err != nil {
		return err
	}
	if st.StartedAt == nil {
		slog.Info("intro course backfill started")
	}

	cursor := uuid.Nil
	if st.LastUserID != nil {
		cursor = *st.LastUserID
	}

	parentClause := ""
	if SkipParentsOnEnroll {
		parentClause = ` AND u.account_type <> 'parent'`
	}

	for {
		rows, err := s.Pool.Query(ctx, `
SELECT u.id
FROM "user".users u
WHERE u.id > $1
  AND u.account_type <> 'system'
  AND u.id <> $3
  AND u.org_id = $4`+parentClause+`
  AND NOT EXISTS (
      SELECT 1 FROM course.course_enrollments ce
      WHERE ce.course_id = $2 AND ce.user_id = u.id AND ce.role = 'student'
  )
ORDER BY u.id
LIMIT $5
`, cursor, courseID, SystemUserID, organization.SeedDefaultOrgID, backfillBatchSize)
		if err != nil {
			return err
		}

		var batch []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		var enrolled int64
		for _, uid := range batch {
			created, err := s.ensureEnrollment(ctx, cfg, s.Pool, uid, PathBackfill)
			if err != nil {
				slog.Warn("intro course backfill enrollment failed", "user_id", uid, "err", err)
			} else if created {
				enrolled++
			}
			cursor = uid
		}
		if err := icrepo.UpdateBackfillProgress(ctx, s.Pool, cursor, enrolled); err != nil {
			return err
		}
		st, _ = icrepo.LoadBackfillState(ctx, s.Pool)
		setBackfillProgress(1)
		remaining, _ := icrepo.CountBackfillRemaining(ctx, s.Pool, courseID, organization.SeedDefaultOrgID, SkipParentsOnEnroll)
		setBackfillRemaining(float64(remaining))
		slog.Debug("intro course backfill batch", "batch_size", len(batch), "enrolled", enrolled, "remaining", remaining)

		if len(batch) < backfillBatchSize {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err := icrepo.MarkBackfillCompleted(ctx, s.Pool, time.Now().UTC()); err != nil {
		return err
	}
	slog.Info("intro course backfill completed")
	setBackfillProgress(1)
	setBackfillRemaining(0)
	return nil
}