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

// Reaction is one user's reaction on a post.
type Reaction struct {
	ID        string
	PostID    string
	UserID    string
	Kind      string
	Value     *float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MyReaction is the viewer's reaction summary for a post.
type MyReaction struct {
	Kind  string   `json:"kind"`
	Value *float64 `json:"value,omitempty"`
}

// PostEngagement aggregates reaction/comment stats for list responses (VC.5 FR-8).
type PostEngagement struct {
	ReactionCount int
	MyReaction    *MyReaction
	AvgStars      *float64
	StarCount     int
	CommentCount  int
	// Grade is set only when the viewer may see the card grade (owner or grader).
	Grade *float64
}

// SetReactionResult describes the outcome of a toggle/set.
type SetReactionResult struct {
	Active   bool
	Reaction *Reaction
	Removed  bool
}

// SetReaction upserts or toggles a reaction for like/vote; sets/updates for star/grade.
// For like/vote with no value change: if already present, removes (toggle off).
func SetReaction(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	userID uuid.UUID,
	kind string,
	value *float64,
) (*SetReactionResult, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if err := validateReactionKindValue(kind, value); err != nil {
		return nil, err
	}
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}

	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM board.post_reactions r
			INNER JOIN board.posts p ON p.id = r.post_id
			INNER JOIN board.boards b ON b.id = p.board_id
			INNER JOIN course.courses c ON c.id = b.course_id
			WHERE c.course_code = $1 AND b.id = $2 AND p.id = $3 AND r.user_id = $4 AND r.kind = $5
		)
	`, courseCode, bid, pid, userID, kind).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists && (kind == ReactionKindLike || kind == ReactionKindVote) {
		// Toggle off.
		if err := DeleteReaction(ctx, pool, courseCode, boardID, postID, userID, kind); err != nil {
			return nil, err
		}
		return &SetReactionResult{Active: false, Removed: true}, nil
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO board.post_reactions (post_id, user_id, kind, value)
		SELECT p.id, $4, $5, $6
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND p.id = $3
		ON CONFLICT (post_id, user_id, kind) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = NOW()
		RETURNING id, post_id, user_id, kind, value, created_at, updated_at
	`, courseCode, bid, pid, userID, kind, value)
	r, err := scanReaction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &SetReactionResult{Active: true, Reaction: &r}, nil
}

// DeleteReaction removes the viewer's reaction of the given kind.
func DeleteReaction(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	userID uuid.UUID,
	kind string,
) error {
	kind = strings.TrimSpace(strings.ToLower(kind))
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM board.post_reactions r
		USING board.posts p, board.boards b, course.courses c
		WHERE r.post_id = p.id AND p.board_id = b.id AND c.id = b.course_id
		  AND c.course_code = $1 AND b.id = $2 AND p.id = $3
		  AND r.user_id = $4 AND r.kind = $5
	`, courseCode, bid, pid, userID, kind)
	return err
}

// GetReaction returns the viewer's reaction of a kind, or nil.
func GetReaction(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	userID uuid.UUID,
	kind string,
) (*Reaction, error) {
	pid, bid, err := parsePostBoardIDs(postID, boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT r.id, r.post_id, r.user_id, r.kind, r.value, r.created_at, r.updated_at
		FROM board.post_reactions r
		INNER JOIN board.posts p ON p.id = r.post_id
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND p.id = $3 AND r.user_id = $4 AND r.kind = $5
	`, courseCode, bid, pid, userID, kind)
	r, err := scanReaction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// LoadPostEngagements returns aggregates for posts on a board for the viewer.
// canGrade: when true, grade values on all posts are visible; otherwise only own posts.
func LoadPostEngagements(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	viewerID uuid.UUID,
	reactionMode string,
	canGrade bool,
) (map[string]PostEngagement, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	kind := ModeToKind(reactionMode)
	out := make(map[string]PostEngagement)

	// Comment counts (visible comments for everyone; managers still count all non-deleted).
	rows, err := pool.Query(ctx, `
		SELECT p.id::text, COUNT(c.id)::int
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c0 ON c0.id = b.course_id
		LEFT JOIN board.post_comments c ON c.post_id = p.id AND c.hidden = FALSE
		WHERE c0.course_code = $1 AND b.id = $2
		GROUP BY p.id
	`, courseCode, bid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var postID string
		var n int
		if err := rows.Scan(&postID, &n); err != nil {
			rows.Close()
			return nil, err
		}
		e := out[postID]
		e.CommentCount = n
		out[postID] = e
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if kind == "" {
		return out, nil
	}

	switch kind {
	case ReactionKindStar:
		rows, err = pool.Query(ctx, `
			SELECT p.id::text,
				COUNT(r.id)::int,
				AVG(r.value)::float8
			FROM board.posts p
			INNER JOIN board.boards b ON b.id = p.board_id
			INNER JOIN course.courses c0 ON c0.id = b.course_id
			LEFT JOIN board.post_reactions r ON r.post_id = p.id AND r.kind = 'star'
			WHERE c0.course_code = $1 AND b.id = $2
			GROUP BY p.id
		`, courseCode, bid)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var postID string
			var n int
			var avg *float64
			if err := rows.Scan(&postID, &n, &avg); err != nil {
				rows.Close()
				return nil, err
			}
			e := out[postID]
			e.ReactionCount = n
			e.StarCount = n
			if n > 0 && avg != nil {
				rounded := float64(int((*avg)*10+0.5)) / 10 // 1 decimal
				e.AvgStars = &rounded
			}
			out[postID] = e
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	case ReactionKindLike, ReactionKindVote:
		rows, err = pool.Query(ctx, `
			SELECT p.id::text, COUNT(r.id)::int
			FROM board.posts p
			INNER JOIN board.boards b ON b.id = p.board_id
			INNER JOIN course.courses c0 ON c0.id = b.course_id
			LEFT JOIN board.post_reactions r ON r.post_id = p.id AND r.kind = $3
			WHERE c0.course_code = $1 AND b.id = $2
			GROUP BY p.id
		`, courseCode, bid, kind)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var postID string
			var n int
			if err := rows.Scan(&postID, &n); err != nil {
				rows.Close()
				return nil, err
			}
			e := out[postID]
			e.ReactionCount = n
			out[postID] = e
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	case ReactionKindGrade:
		// One effective grade per post (most recently updated); visibility filtered below.
		rows, err = pool.Query(ctx, `
			SELECT p.id::text, p.author_id, g.value
			FROM board.posts p
			INNER JOIN board.boards b ON b.id = p.board_id
			INNER JOIN course.courses c0 ON c0.id = b.course_id
			LEFT JOIN LATERAL (
				SELECT r.value
				FROM board.post_reactions r
				WHERE r.post_id = p.id AND r.kind = 'grade'
				ORDER BY r.updated_at DESC
				LIMIT 1
			) g ON TRUE
			WHERE c0.course_code = $1 AND b.id = $2
		`, courseCode, bid)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var postID string
			var authorID uuid.NullUUID
			var val *float64
			if err := rows.Scan(&postID, &authorID, &val); err != nil {
				rows.Close()
				return nil, err
			}
			e := out[postID]
			if val != nil {
				e.ReactionCount = 1
				visible := canGrade || (authorID.Valid && authorID.UUID == viewerID)
				if visible {
					v := *val
					e.Grade = &v
				}
			}
			out[postID] = e
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// Viewer's own reaction (non-grade kinds always; grade only if grader — graders set grades).
	if kind != ReactionKindGrade || canGrade {
		rows, err = pool.Query(ctx, `
			SELECT r.post_id::text, r.kind, r.value
			FROM board.post_reactions r
			INNER JOIN board.posts p ON p.id = r.post_id
			INNER JOIN board.boards b ON b.id = p.board_id
			INNER JOIN course.courses c0 ON c0.id = b.course_id
			WHERE c0.course_code = $1 AND b.id = $2 AND r.user_id = $3 AND r.kind = $4
		`, courseCode, bid, viewerID, kind)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var postID, rk string
			var val *float64
			if err := rows.Scan(&postID, &rk, &val); err != nil {
				rows.Close()
				return nil, err
			}
			e := out[postID]
			e.MyReaction = &MyReaction{Kind: rk, Value: val}
			out[postID] = e
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// EngagementScore returns a numeric score for "most reacted" sorting.
func EngagementScore(e PostEngagement, mode string) float64 {
	switch mode {
	case ReactionModeStar:
		if e.AvgStars == nil {
			return 0
		}
		return *e.AvgStars*1000 + float64(e.StarCount)
	case ReactionModeGrade:
		if e.Grade != nil {
			return *e.Grade
		}
		return float64(e.ReactionCount)
	default:
		return float64(e.ReactionCount)
	}
}

func validateReactionKindValue(kind string, value *float64) error {
	switch kind {
	case ReactionKindLike, ReactionKindVote:
		if value != nil {
			return fmt.Errorf("board: %s reactions do not take a value", kind)
		}
		return nil
	case ReactionKindStar:
		if value == nil || *value < 1 || *value > 5 {
			return fmt.Errorf("board: star rating must be between 1 and 5")
		}
		return nil
	case ReactionKindGrade:
		if value == nil {
			return fmt.Errorf("board: grade value is required")
		}
		return nil
	default:
		return fmt.Errorf("board: invalid reaction kind %q", kind)
	}
}

func parsePostBoardIDs(postID, boardID string) (uuid.UUID, uuid.UUID, error) {
	pid, err := uuid.Parse(postID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return pid, bid, nil
}

func scanReaction(row pgx.Row) (Reaction, error) {
	var (
		id, postID, userID uuid.UUID
		r                  Reaction
		val                *float64
	)
	if err := row.Scan(&id, &postID, &userID, &r.Kind, &val, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return Reaction{}, err
	}
	r.ID = id.String()
	r.PostID = postID.String()
	r.UserID = userID.String()
	r.Value = val
	return r, nil
}
