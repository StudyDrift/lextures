package board

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DailyAnalytics is one precomputed day bucket for a board.
type DailyAnalytics struct {
	BoardID          string    `json:"boardId"`
	Day              time.Time `json:"day"`
	CardCount        int       `json:"cardCount"`
	ContributorCount int       `json:"contributorCount"`
	ReactionCount    int       `json:"reactionCount"`
	CommentCount     int       `json:"commentCount"`
}

// ContributorStat is per-participant contribution counts on a board.
type ContributorStat struct {
	UserID            string `json:"userId"`
	PostCount         int    `json:"postCount"`
	CommentCount      int    `json:"commentCount"`
	ReactionCount     int    `json:"reactionCount"`
	ContributionTotal int    `json:"contributionTotal"`
}

// BoardAnalyticsSummary is the manager-facing analytics payload (VC.10 FR-3).
type BoardAnalyticsSummary struct {
	BoardID            string            `json:"boardId"`
	CardCount          int               `json:"cardCount"`
	UniqueContributors int               `json:"uniqueContributors"`
	ReactionCount      int               `json:"reactionCount"`
	CommentCount       int               `json:"commentCount"`
	LastActivityAt     *time.Time        `json:"lastActivityAt,omitempty"`
	Contributors       []ContributorStat `json:"contributors"`
	Daily              []DailyAnalytics  `json:"daily"`
}

// AdminOverview is the org admin boards dashboard (VC.10 FR-4).
type AdminOverview struct {
	BoardCount            int               `json:"boardCount"`
	ActiveBoardCount      int               `json:"activeBoardCount"`
	CoursesWithBoards     int               `json:"coursesWithBoards"`
	CoursesFeatureEnabled int               `json:"coursesFeatureEnabled"`
	StorageBytes          int64             `json:"storageBytes"`
	TopContentTypes       []ContentTypeStat `json:"topContentTypes"`
	ActiveWindowDays      int               `json:"activeWindowDays"`
}

// ContentTypeStat is a content-type bucket for the admin overview.
type ContentTypeStat struct {
	ContentType string `json:"contentType"`
	Count       int    `json:"count"`
}

// GetBoardAnalytics returns live totals + recent daily rollups for a board.
func GetBoardAnalytics(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string, dailyDays int) (*BoardAnalyticsSummary, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	if dailyDays <= 0 {
		dailyDays = 14
	}

	var exists bool
	var lastActivity time.Time
	err = pool.QueryRow(ctx, `
		SELECT TRUE, b.updated_at
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
	`, courseCode, bid).Scan(&exists, &lastActivity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	sum := &BoardAnalyticsSummary{
		BoardID:        boardID,
		LastActivityAt: &lastActivity,
		Contributors:   []ContributorStat{},
		Daily:          []DailyAnalytics{},
	}

	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM board.posts p WHERE p.board_id = $1 AND p.removed = FALSE),
			(SELECT COUNT(DISTINCT author_id)::int FROM board.posts p
			  WHERE p.board_id = $1 AND p.author_id IS NOT NULL AND p.removed = FALSE),
			(SELECT COUNT(*)::int FROM board.post_reactions r
			  INNER JOIN board.posts p ON p.id = r.post_id WHERE p.board_id = $1 AND p.removed = FALSE),
			(SELECT COUNT(*)::int FROM board.post_comments c
			  INNER JOIN board.posts p ON p.id = c.post_id
			  WHERE p.board_id = $1 AND c.hidden = FALSE AND p.removed = FALSE)
	`, bid).Scan(&sum.CardCount, &sum.UniqueContributors, &sum.ReactionCount, &sum.CommentCount)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		WITH post_counts AS (
			SELECT author_id AS user_id, COUNT(*)::int AS post_count
			FROM board.posts
			WHERE board_id = $1 AND author_id IS NOT NULL AND removed = FALSE
			GROUP BY author_id
		),
		comment_counts AS (
			SELECT c.author_id AS user_id, COUNT(*)::int AS comment_count
			FROM board.post_comments c
			INNER JOIN board.posts p ON p.id = c.post_id
			WHERE p.board_id = $1 AND c.author_id IS NOT NULL AND c.hidden = FALSE AND p.removed = FALSE
			GROUP BY c.author_id
		),
		reaction_counts AS (
			SELECT r.user_id, COUNT(*)::int AS reaction_count
			FROM board.post_reactions r
			INNER JOIN board.posts p ON p.id = r.post_id
			WHERE p.board_id = $1 AND p.removed = FALSE
			GROUP BY r.user_id
		),
		users AS (
			SELECT user_id FROM post_counts
			UNION
			SELECT user_id FROM comment_counts
			UNION
			SELECT user_id FROM reaction_counts
		)
		SELECT u.user_id,
			COALESCE(pc.post_count, 0),
			COALESCE(cc.comment_count, 0),
			COALESCE(rc.reaction_count, 0)
		FROM users u
		LEFT JOIN post_counts pc ON pc.user_id = u.user_id
		LEFT JOIN comment_counts cc ON cc.user_id = u.user_id
		LEFT JOIN reaction_counts rc ON rc.user_id = u.user_id
		ORDER BY (COALESCE(pc.post_count, 0) + COALESCE(cc.comment_count, 0) + COALESCE(rc.reaction_count, 0)) DESC,
		         u.user_id
		LIMIT 100
	`, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid uuid.UUID
		var st ContributorStat
		if err := rows.Scan(&uid, &st.PostCount, &st.CommentCount, &st.ReactionCount); err != nil {
			return nil, err
		}
		st.UserID = uid.String()
		st.ContributionTotal = st.PostCount + st.CommentCount + st.ReactionCount
		sum.Contributors = append(sum.Contributors, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	from := time.Now().UTC().AddDate(0, 0, -dailyDays)
	drows, err := pool.Query(ctx, `
		SELECT board_id, day, card_count, contributor_count, reaction_count, comment_count
		FROM board.analytics_daily
		WHERE board_id = $1 AND day >= $2::date
		ORDER BY day ASC
	`, bid, from)
	if err != nil {
		return nil, err
	}
	defer drows.Close()
	for drows.Next() {
		var d DailyAnalytics
		var id uuid.UUID
		var day time.Time
		if err := drows.Scan(&id, &day, &d.CardCount, &d.ContributorCount, &d.ReactionCount, &d.CommentCount); err != nil {
			return nil, err
		}
		d.BoardID = id.String()
		d.Day = day
		sum.Daily = append(sum.Daily, d)
	}
	return sum, drows.Err()
}

// RefreshAnalyticsDaily upserts today's rollup for all boards (or one board when boardID set).
func RefreshAnalyticsDaily(ctx context.Context, pool *pgxpool.Pool, boardID *uuid.UUID, day time.Time) (int64, error) {
	day = time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	q := `
		INSERT INTO board.analytics_daily (board_id, day, card_count, contributor_count, reaction_count, comment_count)
		SELECT
			b.id,
			$1::date,
			(SELECT COUNT(*)::int FROM board.posts p WHERE p.board_id = b.id AND p.removed = FALSE
			   AND p.created_at::date <= $1::date),
			(SELECT COUNT(DISTINCT p.author_id)::int FROM board.posts p
			  WHERE p.board_id = b.id AND p.author_id IS NOT NULL AND p.removed = FALSE
			    AND p.created_at::date <= $1::date),
			(SELECT COUNT(*)::int FROM board.post_reactions r
			  INNER JOIN board.posts p ON p.id = r.post_id
			  WHERE p.board_id = b.id AND p.removed = FALSE AND r.created_at::date <= $1::date),
			(SELECT COUNT(*)::int FROM board.post_comments c
			  INNER JOIN board.posts p ON p.id = c.post_id
			  WHERE p.board_id = b.id AND c.hidden = FALSE AND p.removed = FALSE AND c.created_at::date <= $1::date)
		FROM board.boards b
	`
	args := []any{day}
	if boardID != nil {
		q += ` WHERE b.id = $2`
		args = append(args, *boardID)
	}
	q += `
		ON CONFLICT (board_id, day) DO UPDATE SET
			card_count = EXCLUDED.card_count,
			contributor_count = EXCLUDED.contributor_count,
			reaction_count = EXCLUDED.reaction_count,
			comment_count = EXCLUDED.comment_count
	`
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// GetAdminOverview aggregates board adoption metrics for an organization.
func GetAdminOverview(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, activeWindowDays int) (AdminOverview, error) {
	if activeWindowDays <= 0 {
		activeWindowDays = 30
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -activeWindowDays)
	out := AdminOverview{ActiveWindowDays: activeWindowDays, TopContentTypes: []ContentTypeStat{}}

	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM board.boards b
			  INNER JOIN course.courses c ON c.id = b.course_id WHERE c.org_id = $1),
			(SELECT COUNT(*)::int FROM board.boards b
			  INNER JOIN course.courses c ON c.id = b.course_id
			  WHERE c.org_id = $1 AND b.updated_at >= $2 AND b.archived = FALSE),
			(SELECT COUNT(DISTINCT b.course_id)::int FROM board.boards b
			  INNER JOIN course.courses c ON c.id = b.course_id WHERE c.org_id = $1),
			(SELECT COUNT(*)::int FROM course.courses c
			  WHERE c.org_id = $1 AND c.visual_boards_enabled = TRUE),
			(SELECT COALESCE(SUM(a.size_bytes), 0)::bigint FROM board.post_attachments a
			  INNER JOIN board.boards b ON b.id = a.board_id
			  INNER JOIN course.courses c ON c.id = b.course_id
			  WHERE c.org_id = $1)
	`, orgID, cutoff).Scan(
		&out.BoardCount, &out.ActiveBoardCount, &out.CoursesWithBoards,
		&out.CoursesFeatureEnabled, &out.StorageBytes,
	)
	if err != nil {
		return out, err
	}

	rows, err := pool.Query(ctx, `
		SELECT p.content_type, COUNT(*)::int
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.org_id = $1 AND p.removed = FALSE
		GROUP BY p.content_type
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`, orgID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var st ContentTypeStat
		if err := rows.Scan(&st.ContentType, &st.Count); err != nil {
			return out, err
		}
		out.TopContentTypes = append(out.TopContentTypes, st)
	}
	return out, rows.Err()
}
