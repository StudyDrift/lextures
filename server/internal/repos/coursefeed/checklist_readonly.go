package coursefeed

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChannelLatest is a feed channel plus its latest root message metadata.
// Read-only — does not call ensureDefaultChannels (CC.1 FR-15).
type ChannelLatest struct {
	ID          uuid.UUID
	Name        string
	SortOrder   int
	LatestAt    *time.Time
	LatestTitle string
}

// ListChannelsWithLatestRoot returns course-level channels and the newest root
// message per channel in one query. Empty when feed channels were never created.
func ListChannelsWithLatestRoot(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]ChannelLatest, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.name, c.sort_order,
       m.created_at AS latest_at,
       COALESCE(m.body, '') AS latest_title
FROM course.feed_channels c
LEFT JOIN LATERAL (
	SELECT fm.created_at, LEFT(fm.body, 120) AS body
	FROM course.feed_messages fm
	WHERE fm.channel_id = c.id AND fm.parent_message_id IS NULL
	ORDER BY fm.created_at DESC
	LIMIT 1
) m ON TRUE
WHERE c.course_id = $1 AND c.group_id IS NULL
ORDER BY c.sort_order ASC, c.created_at ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChannelLatest{}
	for rows.Next() {
		var ch ChannelLatest
		var latestAt *time.Time
		var title string
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.SortOrder, &latestAt, &title); err != nil {
			return nil, err
		}
		ch.LatestAt = latestAt
		ch.LatestTitle = title
		out = append(out, ch)
	}
	return out, rows.Err()
}
