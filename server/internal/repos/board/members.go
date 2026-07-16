package board

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BoardMember is one row in board.board_members.
type BoardMember struct {
	BoardID   string    `json:"boardId"`
	UserID    string    `json:"userId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetMemberRole returns the member role or "" if not a member.
func GetMemberRole(ctx context.Context, pool *pgxpool.Pool, boardID string, userID uuid.UUID) (string, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return "", nil
	}
	var role string
	err = pool.QueryRow(ctx, `
		SELECT role FROM board.board_members WHERE board_id = $1 AND user_id = $2
	`, bid, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

// ListMembers returns members for a board.
func ListMembers(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) ([]BoardMember, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT m.board_id, m.user_id, m.role, m.created_at
		FROM board.board_members m
		INNER JOIN board.boards b ON b.id = m.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		ORDER BY m.created_at ASC
	`, courseCode, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BoardMember, 0)
	for rows.Next() {
		var boardUUID, userUUID uuid.UUID
		var m BoardMember
		if err := rows.Scan(&boardUUID, &userUUID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.BoardID = boardUUID.String()
		m.UserID = userUUID.String()
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpsertMember adds or updates a member. Returns nil,nil if the board is missing.
func UpsertMember(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string, userID uuid.UUID, role string) (*BoardMember, error) {
	norm, err := NormalizeMemberRole(role)
	if err != nil {
		return nil, err
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	// Ensure user is enrolled in the course.
	var enrolled bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM course.course_enrollments e
			INNER JOIN course.courses c ON c.id = e.course_id
			WHERE c.course_code = $1 AND e.user_id = $2 AND e.active
		)
	`, courseCode, userID).Scan(&enrolled)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, fmt.Errorf("board: user is not enrolled in this course")
	}

	var boardUUID, userUUID uuid.UUID
	var m BoardMember
	err = pool.QueryRow(ctx, `
		INSERT INTO board.board_members (board_id, user_id, role)
		SELECT b.id, $3, $4
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		ON CONFLICT (board_id, user_id) DO UPDATE SET role = EXCLUDED.role
		RETURNING board_id, user_id, role, created_at
	`, courseCode, bid, userID, norm).Scan(&boardUUID, &userUUID, &m.Role, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.BoardID = boardUUID.String()
	m.UserID = userUUID.String()
	return &m, nil
}

// RemoveMember deletes a member. Returns true if a row was removed.
func RemoveMember(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, userID string) (bool, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return false, nil
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, nil
	}
	tag, err := pool.Exec(ctx, `
		DELETE FROM board.board_members m
		USING board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE m.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND m.user_id = $3
	`, courseCode, bid, uid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
