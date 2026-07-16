package board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Comment is a threaded comment on a board post.
type Comment struct {
	ID        string
	PostID    string
	ParentID  *string
	AuthorID  *string
	Body      json.RawMessage
	Hidden    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListComments returns comments for a post. When includeHidden is false, hidden rows are omitted.
func ListComments(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	includeHidden bool,
) ([]Comment, error) {
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	q := `
		SELECT c.id, c.post_id, c.parent_id, c.author_id, c.body, c.hidden, c.created_at, c.updated_at
		FROM board.post_comments c
		INNER JOIN board.posts p ON p.id = c.post_id
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c0 ON c0.id = b.course_id
		WHERE c0.course_code = $1 AND b.id = $2 AND p.id = $3`
	if !includeHidden {
		q += ` AND c.hidden = FALSE`
	}
	q += ` ORDER BY c.created_at ASC`

	rows, err := pool.Query(ctx, q, courseCode, bid, pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Comment, 0)
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateComment inserts a sanitized comment, optionally nested under parentID.
func CreateComment(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	authorID uuid.UUID,
	body json.RawMessage,
	parentID *string,
) (*Comment, error) {
	normalized, err := NormalizeBody(body)
	if err != nil {
		return nil, fmt.Errorf("board: invalid body: %w", err)
	}
	if len(normalized) == 0 || string(normalized) == "null" {
		return nil, fmt.Errorf("board: comment body is required")
	}
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	var parentUUID *uuid.UUID
	if parentID != nil && *parentID != "" {
		parsed, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, fmt.Errorf("board: invalid parent_id")
		}
		// Ensure parent belongs to same post.
		var ok bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM board.post_comments c
				INNER JOIN board.posts p ON p.id = c.post_id
				INNER JOIN board.boards b ON b.id = p.board_id
				INNER JOIN course.courses c0 ON c0.id = b.course_id
				WHERE c0.course_code = $1 AND b.id = $2 AND p.id = $3 AND c.id = $4 AND c.hidden = FALSE
			)
		`, courseCode, bid, pid, parsed).Scan(&ok)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("board: parent comment not found")
		}
		parentUUID = &parsed
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO board.post_comments (post_id, parent_id, author_id, body)
		SELECT p.id, $4, $5, $6
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c0 ON c0.id = b.course_id
		WHERE c0.course_code = $1 AND b.id = $2 AND p.id = $3
		RETURNING id, post_id, parent_id, author_id, body, hidden, created_at, updated_at
	`, courseCode, bid, pid, parentUUID, authorID, normalized)
	c, err := scanComment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// PatchCommentInput updates body and/or hidden flag.
type PatchCommentInput struct {
	Body   json.RawMessage
	Hidden *bool
}

// PatchComment updates a comment. Caller must authorize.
func PatchComment(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID, commentID string,
	in PatchCommentInput,
) (*Comment, error) {
	existing, err := GetComment(ctx, pool, courseCode, boardID, postID, commentID)
	if err != nil || existing == nil {
		return existing, err
	}
	body := existing.Body
	if in.Body != nil {
		body, err = NormalizeBody(in.Body)
		if err != nil {
			return nil, fmt.Errorf("board: invalid body: %w", err)
		}
	}
	hidden := existing.Hidden
	if in.Hidden != nil {
		hidden = *in.Hidden
	}
	cid, err := uuid.Parse(commentID)
	if err != nil {
		return nil, nil
	}
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		UPDATE board.post_comments c
		SET body = $5, hidden = $6, updated_at = NOW()
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c0 ON c0.id = b.course_id
		WHERE c.post_id = p.id AND c0.course_code = $1 AND b.id = $2 AND p.id = $3 AND c.id = $4
		RETURNING c.id, c.post_id, c.parent_id, c.author_id, c.body, c.hidden, c.created_at, c.updated_at
	`, courseCode, bid, pid, cid, body, hidden)
	c, err := scanComment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// SoftHideComment marks a comment hidden (audit-preserving delete).
func SoftHideComment(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID, commentID string,
) (*Comment, error) {
	hidden := true
	return PatchComment(ctx, pool, courseCode, boardID, postID, commentID, PatchCommentInput{Hidden: &hidden})
}

// GetComment returns a single comment including hidden ones.
func GetComment(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID, commentID string,
) (*Comment, error) {
	cid, err := uuid.Parse(commentID)
	if err != nil {
		return nil, nil
	}
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT c.id, c.post_id, c.parent_id, c.author_id, c.body, c.hidden, c.created_at, c.updated_at
		FROM board.post_comments c
		INNER JOIN board.posts p ON p.id = c.post_id
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c0 ON c0.id = b.course_id
		WHERE c0.course_code = $1 AND b.id = $2 AND p.id = $3 AND c.id = $4
	`, courseCode, bid, pid, cid)
	c, err := scanComment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// CountRecentCommentsByUser counts comments by a user on a board within a window (rate limit).
func CountRecentCommentsByUser(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	userID uuid.UUID,
	since time.Time,
) (int, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return 0, nil
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM board.post_comments c
		INNER JOIN board.posts p ON p.id = c.post_id
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c0 ON c0.id = b.course_id
		WHERE c0.course_code = $1 AND b.id = $2 AND c.author_id = $3 AND c.created_at >= $4
	`, courseCode, bid, userID, since).Scan(&n)
	return n, err
}

func scanComment(row pgx.Row) (Comment, error) {
	var (
		id, postID            uuid.UUID
		parentID, authorID    uuid.NullUUID
		c                     Comment
		body                  []byte
	)
	if err := row.Scan(&id, &postID, &parentID, &authorID, &body, &c.Hidden, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return Comment{}, err
	}
	c.ID = id.String()
	c.PostID = postID.String()
	if parentID.Valid {
		s := parentID.UUID.String()
		c.ParentID = &s
	}
	if authorID.Valid {
		s := authorID.UUID.String()
		c.AuthorID = &s
	}
	if len(body) > 0 {
		c.Body = json.RawMessage(body)
	}
	return c, nil
}
