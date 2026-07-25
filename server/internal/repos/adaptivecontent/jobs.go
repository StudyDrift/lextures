package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Job status constants (AC.4).
const (
	JobPending    = "pending"
	JobGenerating = "generating"
	JobDone       = "done"
	JobFailed     = "failed"
	JobDeadLetter = "dead_letter"
	JobCanceled   = "canceled"
)

// DefaultJobMaxAttempts is the permanent-failure threshold for transient errors.
const DefaultJobMaxAttempts = 5

// DefaultJobVisibilityTimeout reclaims generating jobs after a worker crash.
const DefaultJobVisibilityTimeout = 10 * time.Minute

// JobRow is a course.adaptive_content_jobs row.
type JobRow struct {
	ID               uuid.UUID
	UnitID           uuid.UUID
	ProfileSignature string
	ContentVersion   int32
	Status           string
	Attempts         int16
	Priority         int16
	RunAfter         time.Time
	LockedBy         *string
	LockedAt         *time.Time
	LastError        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// EnqueueJobParams describes a generation job to insert (ids only — no PII).
type EnqueueJobParams struct {
	UnitID           uuid.UUID
	ProfileSignature string
	ContentVersion   int32
	Priority         int16 // higher runs first; 0 default
	RunAfter         time.Time
}

// EnqueueJob inserts a pending job. Dedupe unique index returns the existing
// in-flight/done job id without creating a second row (AC.4 FR-2).
// Returns (id, created, error). created=false when deduped against existing.
func EnqueueJob(ctx context.Context, pool *pgxpool.Pool, p EnqueueJobParams) (id uuid.UUID, created bool, err error) {
	if pool == nil {
		return uuid.Nil, false, errors.New("adaptivecontent jobs: nil pool")
	}
	if p.UnitID == uuid.Nil || p.ProfileSignature == "" {
		return uuid.Nil, false, errors.New("adaptivecontent jobs: unit_id and profile_signature required")
	}
	if p.ContentVersion <= 0 {
		p.ContentVersion = 1
	}
	runAfter := p.RunAfter
	if runAfter.IsZero() {
		runAfter = time.Now().UTC()
	}

	err = pool.QueryRow(ctx, `
INSERT INTO course.adaptive_content_jobs (
  unit_id, profile_signature, content_version, status, priority, run_after
) VALUES ($1, $2, $3, 'pending', $4, $5)
ON CONFLICT (unit_id, profile_signature, content_version)
  WHERE status IN ('pending','generating','done')
DO NOTHING
RETURNING id
`, p.UnitID, p.ProfileSignature, p.ContentVersion, p.Priority, runAfter).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Deduped — return existing in-flight/done job.
		qerr := pool.QueryRow(ctx, `
SELECT id FROM course.adaptive_content_jobs
WHERE unit_id = $1 AND profile_signature = $2 AND content_version = $3
  AND status IN ('pending','generating','done')
ORDER BY created_at ASC
LIMIT 1
`, p.UnitID, p.ProfileSignature, p.ContentVersion).Scan(&id)
		return id, false, qerr
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

// ClaimJobs leases up to limit ready pending jobs (and reclaims stale generating ones)
// using SELECT … FOR UPDATE SKIP LOCKED.
func ClaimJobs(ctx context.Context, pool *pgxpool.Pool, workerID string, limit int, now time.Time, visibility time.Duration) ([]JobRow, error) {
	if pool == nil {
		return nil, errors.New("adaptivecontent jobs: nil pool")
	}
	if limit <= 0 {
		limit = 1
	}
	if visibility <= 0 {
		visibility = DefaultJobVisibilityTimeout
	}
	if workerID == "" {
		workerID = "worker"
	}
	staleBefore := now.Add(-visibility)

	rows, err := pool.Query(ctx, `
WITH claimed AS (
    SELECT id
    FROM course.adaptive_content_jobs
    WHERE (status = 'pending' AND run_after <= $1)
       OR (status = 'generating' AND locked_at IS NOT NULL AND locked_at < $2)
    ORDER BY priority DESC, run_after ASC
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
UPDATE course.adaptive_content_jobs j
SET status = 'generating',
    locked_by = $4,
    locked_at = $1,
    attempts = j.attempts + 1,
    updated_at = $1
FROM claimed
WHERE j.id = claimed.id
RETURNING j.id, j.unit_id, j.profile_signature, j.content_version, j.status, j.attempts,
          j.priority, j.run_after, j.locked_by, j.locked_at, j.last_error, j.created_at, j.updated_at
`, now, staleBefore, limit, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobRow
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// CompleteJob marks a job done (successful generation or permanent non-retry).
func CompleteJob(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, now time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE course.adaptive_content_jobs
SET status = 'done', locked_by = NULL, locked_at = NULL, last_error = NULL, updated_at = $2
WHERE id = $1
`, id, now)
	return err
}

// CancelJob marks a job canceled (e.g. budget exhausted) so it can be re-enqueued later.
func CancelJob(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, reason string, now time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE course.adaptive_content_jobs
SET status = 'canceled', locked_by = NULL, locked_at = NULL, last_error = $2, updated_at = $3
WHERE id = $1
`, id, reason, now)
	return err
}

// FailJob records a transient failure. If attempts < maxAttempts the job returns
// to pending with exponential backoff; otherwise it is dead-lettered.
func FailJob(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, attempts int16, maxAttempts int, errMsg string, now time.Time, backoff time.Duration) (deadLetter bool, err error) {
	if maxAttempts <= 0 {
		maxAttempts = DefaultJobMaxAttempts
	}
	if int(attempts) >= maxAttempts {
		_, err = pool.Exec(ctx, `
UPDATE course.adaptive_content_jobs
SET status = 'dead_letter', locked_by = NULL, locked_at = NULL, last_error = $2, updated_at = $3
WHERE id = $1
`, id, errMsg, now)
		return true, err
	}
	runAfter := now.Add(backoff)
	_, err = pool.Exec(ctx, `
UPDATE course.adaptive_content_jobs
SET status = 'pending', locked_by = NULL, locked_at = NULL, last_error = $2,
    run_after = $3, updated_at = $4
WHERE id = $1
`, id, errMsg, runAfter, now)
	return false, err
}

// ReleaseJobToPending puts a claimed job back to pending without counting a failure
// (used when platform/course generation is paused).
func ReleaseJobToPending(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, runAfter time.Time, now time.Time) error {
	// Decrement the attempt that ClaimJobs just added so pause doesn't burn retries.
	_, err := pool.Exec(ctx, `
UPDATE course.adaptive_content_jobs
SET status = 'pending',
    locked_by = NULL,
    locked_at = NULL,
    attempts = GREATEST(attempts - 1, 0),
    run_after = $2,
    updated_at = $3
WHERE id = $1
`, id, runAfter, now)
	return err
}

// CountPendingJobs returns the number of pending (ready or delayed) jobs.
func CountPendingJobs(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.adaptive_content_jobs WHERE status = 'pending'
`).Scan(&n)
	return n, err
}

// CountGeneratingJobs returns in-flight generation jobs.
func CountGeneratingJobs(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.adaptive_content_jobs WHERE status = 'generating'
`).Scan(&n)
	return n, err
}

// HasInFlightOrDoneJob reports whether a non-retriable job exists for the triple.
func HasInFlightOrDoneJob(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, signature string, contentVersion int32) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.adaptive_content_jobs
  WHERE unit_id = $1 AND profile_signature = $2 AND content_version = $3
    AND status IN ('pending','generating','done')
)
`, unitID, signature, contentVersion).Scan(&ok)
	return ok, err
}

// HasReadyVariant reports whether a serveable variant exists for (unit, signature, version).
func HasReadyVariant(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, signature string, contentVersion int32) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.content_variants
  WHERE unit_id = $1 AND profile_signature = $2
    AND content_version = $3
    AND status IN ('approved','auto_served')
)
`, unitID, signature, contentVersion).Scan(&ok)
	return ok, err
}

// GetUnitCourseID returns the course_id for a unit (for job processing without course scope).
func GetUnitByID(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) (*UnitRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity
FROM course.adaptive_content_units
WHERE id = $1
`, unitID)
	u, err := scanUnit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetAnyProfileBySignature returns one stored profile for (unit, signature) for generation inputs.
func GetAnyProfileBySignature(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, signature string) (*ProfileRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, enrollment_id, user_id, profile_signature, emphasis_mode, payload_json,
       source_attempt_id, target_bloom::text, reading_level_pref, modality_pref, axis_set,
       is_neutral, created_at
FROM course.adaptation_profiles
WHERE unit_id = $1 AND profile_signature = $2
ORDER BY created_at DESC
LIMIT 1
`, unitID, signature)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// ListTopSignatures returns up to limit signatures ordered by cohort frequency (desc).
func ListTopSignatures(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, limit int) ([]SignatureCount, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := pool.Query(ctx, `
SELECT profile_signature, COALESCE(emphasis_mode, 'unknown'), COUNT(*)::bigint
FROM course.adaptation_profiles
WHERE unit_id = $1 AND is_neutral = FALSE AND profile_signature <> 'base'
GROUP BY profile_signature, emphasis_mode
ORDER BY COUNT(*) DESC, profile_signature ASC
LIMIT $2
`, unitID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SignatureCount
	for rows.Next() {
		var s SignatureCount
		if err := rows.Scan(&s.ProfileSignature, &s.EmphasisMode, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []SignatureCount{}
	}
	return out, rows.Err()
}

// ListSignaturesNeedingRegen returns distinct non-neutral signatures that lack a ready
// variant for the unit's current content_version (for post-edit re-enqueue).
func ListSignaturesNeedingRegen(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, contentVersion int32, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := pool.Query(ctx, `
SELECT p.profile_signature
FROM (
  SELECT profile_signature, COUNT(*) AS c
  FROM course.adaptation_profiles
  WHERE unit_id = $1 AND is_neutral = FALSE AND profile_signature <> 'base'
  GROUP BY profile_signature
) p
WHERE NOT EXISTS (
  SELECT 1 FROM course.content_variants v
  WHERE v.unit_id = $1
    AND v.profile_signature = p.profile_signature
    AND v.content_version = $2
    AND v.status IN ('approved','auto_served','pending_review')
)
ORDER BY p.c DESC, p.profile_signature ASC
LIMIT $3
`, unitID, contentVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []string{}
	}
	return out, rows.Err()
}

func scanJob(row scannable) (JobRow, error) {
	var j JobRow
	err := row.Scan(
		&j.ID, &j.UnitID, &j.ProfileSignature, &j.ContentVersion, &j.Status, &j.Attempts,
		&j.Priority, &j.RunAfter, &j.LockedBy, &j.LockedAt, &j.LastError, &j.CreatedAt, &j.UpdatedAt,
	)
	return j, err
}
