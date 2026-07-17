package quizgame

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GuestRetentionDays returns how long guest player rows are kept (faster than enrolled).
func GuestRetentionDays(platformRetentionDays int) int {
	if platformRetentionDays <= 0 {
		platformRetentionDays = DefaultRetentionDays
	}
	if platformRetentionDays < DefaultGuestRetentionDays {
		return platformRetentionDays
	}
	return DefaultGuestRetentionDays
}

// RetentionResult summarises a retention pass.
type RetentionResult struct {
	GuestPlayersPurged int `json:"guestPlayersPurged"`
	ResponsesAnonymised int `json:"responsesAnonymised"`
	SessionsTouched    int `json:"sessionsTouched"`
}

// RunRetention anonymises/deletes aged quizgame data (IQ.11 FR-5).
// Guest players (and their responses) older than guestCutoff are purged.
// Enrolled player responses older than enrolledCutoff have answer payloads redacted.
func RunRetention(ctx context.Context, pool *pgxpool.Pool, guestCutoff, enrolledCutoff time.Time, limit int) (RetentionResult, error) {
	if limit <= 0 {
		limit = 200
	}
	var out RetentionResult

	// Purge guest players (and cascade responses) whose sessions ended before guestCutoff.
	tag, err := pool.Exec(ctx, `
		DELETE FROM quizgame.session_players p
		USING quizgame.sessions s
		WHERE p.session_id = s.id
		  AND p.user_id IS NULL
		  AND s.status IN ('ended', 'abandoned')
		  AND COALESCE(s.ended_at, s.created_at) < $1
		  AND p.id IN (
			SELECT p2.id
			FROM quizgame.session_players p2
			INNER JOIN quizgame.sessions s2 ON s2.id = p2.session_id
			WHERE p2.user_id IS NULL
			  AND s2.status IN ('ended', 'abandoned')
			  AND COALESCE(s2.ended_at, s2.created_at) < $1
			ORDER BY COALESCE(s2.ended_at, s2.created_at) ASC
			LIMIT $2
		  )
	`, guestCutoff, limit)
	if err != nil {
		return out, err
	}
	out.GuestPlayersPurged = int(tag.RowsAffected())

	// Anonymise enrolled responses past the retention window (keep scores, redact answer body).
	tag, err = pool.Exec(ctx, `
		UPDATE quizgame.session_responses r
		SET answer = '{"redacted":true}'::jsonb
		FROM quizgame.sessions s
		WHERE r.session_id = s.id
		  AND s.status IN ('ended', 'abandoned')
		  AND COALESCE(s.ended_at, s.created_at) < $1
		  AND r.answer IS DISTINCT FROM '{"redacted":true}'::jsonb
		  AND r.session_id IN (
			SELECT s2.id
			FROM quizgame.sessions s2
			WHERE s2.status IN ('ended', 'abandoned')
			  AND COALESCE(s2.ended_at, s2.created_at) < $1
			ORDER BY COALESCE(s2.ended_at, s2.created_at) ASC
			LIMIT $2
		  )
	`, enrolledCutoff, limit)
	if err != nil {
		return out, err
	}
	out.ResponsesAnonymised = int(tag.RowsAffected())

	// Clear guest nicknames on aged enrolled-adjacent rows that remain (defence in depth).
	_, _ = pool.Exec(ctx, `
		UPDATE quizgame.session_players p
		SET nickname = 'Player'
		FROM quizgame.sessions s
		WHERE p.session_id = s.id
		  AND p.user_id IS NULL
		  AND s.status IN ('ended', 'abandoned')
		  AND COALESCE(s.ended_at, s.created_at) < $1
		  AND p.nickname <> 'Player'
	`, guestCutoff)

	var touched int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM quizgame.sessions
		WHERE status IN ('ended', 'abandoned')
		  AND COALESCE(ended_at, created_at) < $1
	`, enrolledCutoff).Scan(&touched)
	out.SessionsTouched = touched
	return out, nil
}

// BulkArchiveKits archives kits older than cutoff (by updated_at) for an org.
func BulkArchiveKits(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, olderThan time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.kits k
		SET archived = TRUE, updated_at = NOW()
		FROM course.courses c
		WHERE c.id = k.course_id
		  AND c.org_id = $1
		  AND k.archived = FALSE
		  AND k.is_template = FALSE
		  AND k.updated_at < $2
		  AND k.id IN (
			SELECT k2.id
			FROM quizgame.kits k2
			INNER JOIN course.courses c2 ON c2.id = k2.course_id
			WHERE c2.org_id = $1
			  AND k2.archived = FALSE
			  AND k2.is_template = FALSE
			  AND k2.updated_at < $2
			ORDER BY k2.updated_at ASC
			LIMIT $3
		  )
	`, orgID, olderThan, limit)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
