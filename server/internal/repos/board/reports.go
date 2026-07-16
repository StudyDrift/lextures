package board

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ReportKindUser      = "user"
	ReportKindFilter    = "filter"
	ReportKindAVBlocked = "av_blocked"

	ReportStatusOpen      = "open"
	ReportStatusResolved  = "resolved"
	ReportStatusDismissed = "dismissed"
)

// Report is a moderation queue item.
type Report struct {
	ID         string
	BoardID    string
	PostID     *string
	CommentID  *string
	ReporterID *string
	Reason     string
	Kind       string
	Status     string
	CreatedAt  time.Time
	ResolvedAt *time.Time
	ResolvedBy *string
}

func scanReport(row pgx.Row) (Report, error) {
	var (
		id, boardID           uuid.UUID
		postID, commentID     uuid.NullUUID
		reporterID, resolvedBy uuid.NullUUID
		r                     Report
		resolvedAt            *time.Time
	)
	if err := row.Scan(
		&id, &boardID, &postID, &commentID, &reporterID, &r.Reason, &r.Kind, &r.Status,
		&r.CreatedAt, &resolvedAt, &resolvedBy,
	); err != nil {
		return Report{}, err
	}
	r.ID = id.String()
	r.BoardID = boardID.String()
	if postID.Valid {
		s := postID.UUID.String()
		r.PostID = &s
	}
	if commentID.Valid {
		s := commentID.UUID.String()
		r.CommentID = &s
	}
	if reporterID.Valid {
		s := reporterID.UUID.String()
		r.ReporterID = &s
	}
	r.ResolvedAt = resolvedAt
	if resolvedBy.Valid {
		s := resolvedBy.UUID.String()
		r.ResolvedBy = &s
	}
	return r, nil
}

// CreateReport inserts a report for a post or comment. Duplicate open user reports are ignored (dedupe).
func CreateReport(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	reporterID *uuid.UUID,
	postID, commentID *string,
	reason, kind string,
) (*Report, error) {
	if (postID == nil || *postID == "") && (commentID == nil || *commentID == "") {
		return nil, fmt.Errorf("board: postId or commentId is required")
	}
	if kind == "" {
		kind = ReportKindUser
	}
	switch kind {
	case ReportKindUser, ReportKindFilter, ReportKindAVBlocked:
	default:
		return nil, fmt.Errorf("board: invalid report kind")
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	var postUUID, commentUUID *uuid.UUID
	if postID != nil && strings.TrimSpace(*postID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*postID))
		if err != nil {
			return nil, fmt.Errorf("board: invalid post_id")
		}
		postUUID = &parsed
	}
	if commentID != nil && strings.TrimSpace(*commentID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*commentID))
		if err != nil {
			return nil, fmt.Errorf("board: invalid comment_id")
		}
		commentUUID = &parsed
	}
	// Ensure target belongs to board.
	if postUUID != nil {
		var ok bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM board.posts p
				INNER JOIN board.boards b ON b.id = p.board_id
				INNER JOIN course.courses c ON c.id = b.course_id
				WHERE c.course_code = $1 AND b.id = $2 AND p.id = $3
			)
		`, courseCode, bid, *postUUID).Scan(&ok)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("board: post not found")
		}
	}
	if commentUUID != nil {
		var ok bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM board.post_comments pc
				INNER JOIN board.posts p ON p.id = pc.post_id
				INNER JOIN board.boards b ON b.id = p.board_id
				INNER JOIN course.courses c ON c.id = b.course_id
				WHERE c.course_code = $1 AND b.id = $2 AND pc.id = $3
			)
		`, courseCode, bid, *commentUUID).Scan(&ok)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("board: comment not found")
		}
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO board.reports (board_id, post_id, comment_id, reporter_id, reason, kind)
		SELECT b.id, $3, $4, $5, $6, $7
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		RETURNING id, board_id, post_id, comment_id, reporter_id, reason, kind, status, created_at, resolved_at, resolved_by
	`, courseCode, bid, postUUID, commentUUID, reporterID, strings.TrimSpace(reason), kind)
	rep, err := scanReport(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Duplicate open user report — return existing.
			return GetOpenReport(ctx, pool, courseCode, boardID, postUUID, commentUUID, reporterID)
		}
		return nil, err
	}
	return &rep, nil
}

// GetOpenReport finds an open report for dedupe.
func GetOpenReport(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	postID, commentID *uuid.UUID,
	reporterID *uuid.UUID,
) (*Report, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT r.id, r.board_id, r.post_id, r.comment_id, r.reporter_id, r.reason, r.kind, r.status,
			r.created_at, r.resolved_at, r.resolved_by
		FROM board.reports r
		INNER JOIN board.boards b ON b.id = r.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND r.status = 'open'
		  AND (($3::uuid IS NOT NULL AND r.post_id = $3) OR ($4::uuid IS NOT NULL AND r.comment_id = $4))
		  AND (($5::uuid IS NULL AND r.reporter_id IS NULL) OR r.reporter_id = $5)
		ORDER BY r.created_at DESC
		LIMIT 1
	`, courseCode, bid, postID, commentID, reporterID)
	rep, err := scanReport(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rep, nil
}

// ListOpenReports returns open reports for the moderation queue.
func ListOpenReports(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) ([]Report, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT r.id, r.board_id, r.post_id, r.comment_id, r.reporter_id, r.reason, r.kind, r.status,
			r.created_at, r.resolved_at, r.resolved_by
		FROM board.reports r
		INNER JOIN board.boards b ON b.id = r.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND r.status = 'open'
		ORDER BY r.created_at ASC
	`, courseCode, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Report, 0)
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

// ResolveReport marks a report resolved or dismissed.
func ResolveReport(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, reportID string,
	resolver uuid.UUID,
	status string,
) (*Report, error) {
	switch status {
	case ReportStatusResolved, ReportStatusDismissed:
	default:
		return nil, fmt.Errorf("board: invalid report status")
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rid, err := uuid.Parse(reportID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE board.reports r
		SET status = $4, resolved_at = NOW(), resolved_by = $5
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE r.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND r.id = $3 AND r.status = 'open'
		RETURNING r.id, r.board_id, r.post_id, r.comment_id, r.reporter_id, r.reason, r.kind, r.status,
			r.created_at, r.resolved_at, r.resolved_by
	`, courseCode, bid, rid, status, resolver)
	rep, err := scanReport(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rep, nil
}

// CountRecentReportsByReporter enforces per-user rate limiting.
func CountRecentReportsByReporter(ctx context.Context, pool *pgxpool.Pool, boardID string, reporter uuid.UUID, since time.Time) (int, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return 0, nil
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM board.reports
		WHERE board_id = $1 AND reporter_id = $2 AND created_at >= $3 AND kind = 'user'
	`, bid, reporter, since).Scan(&n)
	return n, err
}
