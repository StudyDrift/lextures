package discussions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdatePostBody replaces a post body (TipTap JSON) and bumps updated_at.
func UpdatePostBody(ctx context.Context, pool *pgxpool.Pool, postID uuid.UUID, body json.RawMessage) (*PostRow, error) {
	if !json.Valid(body) {
		return nil, errors.New("invalid post body json")
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil || root == nil {
		return nil, errors.New("invalid post body")
	}
	if ty, _ := root["type"].(string); ty != "doc" {
		return nil, errors.New("post body must be a TipTap doc")
	}
	var r PostRow
	var parent *uuid.UUID
	var bodyOut []byte
	err := pool.QueryRow(ctx, `
UPDATE course.discussion_posts
SET body = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, thread_id, parent_post_id, author_id, body, upvote_count, created_at, updated_at
`, postID, body).Scan(
		&r.ID, &r.ThreadID, &parent, &r.AuthorID, &bodyOut, &r.UpvoteCount, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	r.ParentPostID = parent
	r.Body = json.RawMessage(bodyOut)
	return &r, nil
}

// FindForumByName returns a forum by exact name within a course, if any.
func FindForumByName(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, name string) (*ForumRow, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var r ForumRow
	var desc sql.NullString
	err := pool.QueryRow(ctx, `
SELECT id, course_id, name, description, position, created_at
FROM course.discussion_forums
WHERE course_id = $1 AND name = $2
LIMIT 1
`, courseID, name).Scan(&r.ID, &r.CourseID, &r.Name, &desc, &r.Position, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if desc.Valid {
		s := desc.String
		r.Description = &s
	}
	return &r, nil
}

// FindThreadByTitle returns a thread by exact title within a forum, if any.
func FindThreadByTitle(ctx context.Context, pool *pgxpool.Pool, forumID uuid.UUID, title string) (*ThreadDetail, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
SELECT
  t.id, t.forum_id, t.assignment_structure_item_id, t.author_id, t.title, t.body,
  t.is_pinned, t.is_locked, t.require_post_first, t.created_at, t.updated_at,
  COALESCE((SELECT COUNT(*)::int FROM course.discussion_posts p WHERE p.thread_id = t.id), 0)
FROM course.discussion_threads t
WHERE t.forum_id = $1 AND t.title = $2
LIMIT 1
`, forumID, title)
	var d ThreadDetail
	var assign *uuid.UUID
	var body []byte
	if err := row.Scan(
		&d.ID, &d.ForumID, &assign, &d.AuthorID, &d.Title, &body,
		&d.IsPinned, &d.IsLocked, &d.RequirePostFirst, &d.CreatedAt, &d.UpdatedAt,
		&d.ReplyCount,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d.AssignmentStructureItemID = assign
	d.Body = json.RawMessage(body)
	return &d, nil
}

// ListPostsOrdered returns posts with optional newest-first ordering (flat list).
func ListPostsOrdered(ctx context.Context, pool *pgxpool.Pool, threadID, viewerID uuid.UUID, staff, hidePeers, newestFirst bool, limit, offset int) ([]PostRow, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	order := `ORDER BY p.created_at ASC, p.id ASC`
	if newestFirst {
		order = `ORDER BY p.created_at DESC, p.id DESC`
	}
	countQ := `
SELECT COUNT(*)::int FROM course.discussion_posts p
WHERE p.thread_id = $1
`
	countArgs := []any{threadID}
	if hidePeers && !staff {
		countQ += ` AND p.author_id = $2`
		countArgs = append(countArgs, viewerID)
	}
	var total int
	if err := pool.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `
SELECT p.id, p.thread_id, p.parent_post_id, p.author_id, p.body, p.upvote_count, p.created_at, p.updated_at,
       EXISTS(SELECT 1 FROM course.discussion_post_upvotes u WHERE u.post_id = p.id AND u.user_id = $2)
FROM course.discussion_posts p
WHERE p.thread_id = $1
`
	args := []any{threadID, viewerID, limit, offset}
	if hidePeers && !staff {
		q += ` AND p.author_id = $2`
	}
	q += order + `
LIMIT $3 OFFSET $4
`
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []PostRow
	for rows.Next() {
		var r PostRow
		var parent *uuid.UUID
		var body []byte
		if err := rows.Scan(&r.ID, &r.ThreadID, &parent, &r.AuthorID, &body, &r.UpvoteCount, &r.CreatedAt, &r.UpdatedAt, &r.ViewerUpvoted); err != nil {
			return nil, 0, err
		}
		r.ParentPostID = parent
		r.Body = json.RawMessage(body)
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// SoftDeletePostsByAuthor marks posts by replacing body with a tombstone TipTap doc via callback.
// Prefer UpdatePostBody from callers that own TipTap encoding.
func ListPostIDsByAuthor(ctx context.Context, pool *pgxpool.Pool, threadID, authorID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT id FROM course.discussion_posts
WHERE thread_id = $1 AND author_id = $2
ORDER BY created_at ASC
`, threadID, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// IsContentToolForumName reports whether a forum is reserved for content-tool threads.
func IsContentToolForumName(name string) bool {
	return strings.HasPrefix(strings.TrimSpace(name), "__ct_")
}
