package discussions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetPost returns a post row if it exists in the course, with viewerUpvoted when viewer is non-nil.
func GetPost(ctx context.Context, pool *pgxpool.Pool, courseID, postID uuid.UUID, viewer *uuid.UUID) (*PostRow, error) {
	upvoteSel := `false`
	args := []any{postID, courseID}
	if viewer != nil {
		upvoteSel = `EXISTS(SELECT 1 FROM course.discussion_post_upvotes u WHERE u.post_id = p.id AND u.user_id = $3)`
		args = []any{postID, courseID, *viewer}
	}
	row := pool.QueryRow(ctx, `
SELECT p.id, p.thread_id, p.parent_post_id, p.author_id, p.body, p.upvote_count, p.created_at, p.updated_at,
       `+upvoteSel+`,
       `+authorColumns+`
FROM course.discussion_posts p
INNER JOIN course.discussion_threads t ON t.id = p.thread_id
INNER JOIN course.discussion_forums f ON f.id = t.forum_id
`+postAuthorJoin+`
WHERE p.id = $1 AND f.course_id = $2
`, args...)
	var r PostRow
	var parent *uuid.UUID
	var body []byte
	var name, avatar sql.NullString
	if err := row.Scan(&r.ID, &r.ThreadID, &parent, &r.AuthorID, &body, &r.UpvoteCount, &r.CreatedAt, &r.UpdatedAt, &r.ViewerUpvoted, &name, &avatar); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	r.ParentPostID = parent
	r.Body = json.RawMessage(body)
	r.AuthorDisplayName, r.AuthorAvatarURL = authorFromNulls(name, avatar)
	return &r, nil
}
