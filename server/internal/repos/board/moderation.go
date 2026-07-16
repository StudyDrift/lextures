package board

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ModerationOpen     = "open"
	ModerationApproval = "approval"

	FilterBlock = "block"
	FilterFlag  = "flag"

	PostStatusApproved = "approved"
	PostStatusPending  = "pending"
	PostStatusRejected = "rejected"

	ModActionApprove       = "approve"
	ModActionReject        = "reject"
	ModActionHide          = "hide"
	ModActionUnhide        = "unhide"
	ModActionRemove        = "remove"
	ModActionLock          = "lock"
	ModActionUnlock        = "unlock"
	ModActionFreeze        = "freeze"
	ModActionUnfreeze      = "unfreeze"
	ModActionFilterHit     = "filter_hit"
	ModActionReportResolve = "report_resolve"
	ModActionAVBlocked     = "av_blocked"

	TargetPost    = "post"
	TargetComment = "comment"
	TargetBoard   = "board"
	TargetReport  = "report"
)

// NormalizeModerationMode returns open|approval.
func NormalizeModerationMode(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ModerationOpen:
		return ModerationOpen, nil
	case ModerationApproval:
		return ModerationApproval, nil
	default:
		return "", fmt.Errorf("board: invalid moderation_mode")
	}
}

// NormalizeFilterAction returns block|flag.
func NormalizeFilterAction(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case FilterBlock:
		return FilterBlock, nil
	case FilterFlag:
		return FilterFlag, nil
	default:
		return "", fmt.Errorf("board: invalid filter_action")
	}
}

// ApplyMinorsModerationFloor forces approval + block when the course has enrolled minors
// and COPPA workflow is enabled (org floor; instructors cannot loosen below it).
func ApplyMinorsModerationFloor(mode, filterAction string, minorsFloor bool) (string, string) {
	if !minorsFloor {
		return mode, filterAction
	}
	return ModerationApproval, FilterBlock
}

// ModerationLogEntry is one append-only audit row.
type ModerationLogEntry struct {
	ID         int64
	BoardID    string
	ActorID    *string
	Action     string
	TargetType string
	TargetID   *string
	Reason     string
	CreatedAt  time.Time
}

// InsertModerationLog appends an audit entry. Failures are returned to the caller.
func InsertModerationLog(
	ctx context.Context,
	pool *pgxpool.Pool,
	boardID string,
	actorID *uuid.UUID,
	action, targetType string,
	targetID *uuid.UUID,
	reason string,
) error {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return fmt.Errorf("board: invalid board_id")
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO board.moderation_log (board_id, actor_id, action, target_type, target_id, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, bid, actorID, action, targetType, targetID, strings.TrimSpace(reason))
	return err
}

// ListModerationLog returns recent audit entries for a board (newest first).
func ListModerationLog(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string, limit int) ([]ModerationLogEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT m.id, m.board_id, m.actor_id, m.action, m.target_type, m.target_id, m.reason, m.created_at
		FROM board.moderation_log m
		INNER JOIN board.boards b ON b.id = m.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $3
	`, courseCode, bid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ModerationLogEntry, 0)
	for rows.Next() {
		var e ModerationLogEntry
		var boardUUID uuid.UUID
		var actor, target uuid.NullUUID
		if err := rows.Scan(&e.ID, &boardUUID, &actor, &e.Action, &e.TargetType, &target, &e.Reason, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.BoardID = boardUUID.String()
		if actor.Valid {
			s := actor.UUID.String()
			e.ActorID = &s
		}
		if target.Valid {
			s := target.UUID.String()
			e.TargetID = &s
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SetPostStatus updates moderation status (approve/reject).
func SetPostStatus(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, postID, status string) (*Post, error) {
	switch status {
	case PostStatusApproved, PostStatusPending, PostStatusRejected:
	default:
		return nil, fmt.Errorf("board: invalid post status")
	}
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE board.posts p
		SET status = $4, updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.id = $3
		RETURNING `+selectPostCols()+`
	`, courseCode, bid, pid, status)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	list := []Post{p}
	if err := attachAttachments(ctx, pool, courseCode, boardID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// SetPostHidden toggles soft-hide (reversible).
func SetPostHidden(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, postID string, hidden bool) (*Post, error) {
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE board.posts p
		SET hidden = $4, updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.id = $3
		  AND p.removed = FALSE
		RETURNING `+selectPostCols()+`
	`, courseCode, bid, pid, hidden)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	list := []Post{p}
	if err := attachAttachments(ctx, pool, courseCode, boardID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// SoftRemovePost marks a post removed (soft-delete; retained for audit).
func SoftRemovePost(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, postID string) (*Post, error) {
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE board.posts p
		SET removed = TRUE, hidden = TRUE, updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.id = $3
		RETURNING `+selectPostCols()+`
	`, courseCode, bid, pid)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	list := []Post{p}
	if err := attachAttachments(ctx, pool, courseCode, boardID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// ListPendingPosts returns posts awaiting approval.
func ListPendingPosts(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) ([]Post, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT `+selectPostCols()+`
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND p.status = $3 AND p.removed = FALSE
		ORDER BY p.created_at ASC
	`, courseCode, bid, PostStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := attachAttachments(ctx, pool, courseCode, boardID, out); err != nil {
		return nil, err
	}
	return out, nil
}

// PostVisibleToViewer reports whether a post should appear on the board surface for the viewer.
func PostVisibleToViewer(p Post, viewerID string, isManager bool) bool {
	if isManager {
		return !p.Removed // managers still see hidden/pending for queue; surface may filter separately
	}
	if p.Removed || p.Hidden {
		return false
	}
	if p.Status == PostStatusApproved {
		return true
	}
	if p.Status == PostStatusPending && p.AuthorID != nil && *p.AuthorID == viewerID {
		return true
	}
	return false
}

// FilterVisiblePosts keeps posts the viewer may see on the board surface.
// Managers see approved + pending (not rejected/removed/hidden); peers see approved only (+ own pending).
func FilterVisiblePosts(posts []Post, viewerID string, isManager bool) []Post {
	out := make([]Post, 0, len(posts))
	for _, p := range posts {
		if isManager {
			if p.Removed || p.Hidden || p.Status == PostStatusRejected {
				continue
			}
			out = append(out, p)
			continue
		}
		if PostVisibleToViewer(p, viewerID, false) {
			out = append(out, p)
		}
	}
	return out
}
