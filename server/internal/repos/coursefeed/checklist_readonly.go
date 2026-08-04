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

// StaffWelcome is a staff-authored announcements root message of sufficient length.
type StaffWelcome struct {
	BodyLen       int
	AuthorIsStaff bool
	PostedAt      *time.Time
}

// ChecklistChannel is a feed channel with optional staff-welcome detection for announcements.
type ChecklistChannel struct {
	ChannelLatest
	StaffWelcome *StaffWelcome
}

// ListChecklistFeedChannels returns course-level channels, latest root message,
// and for the announcements channel whether a staff welcome (≥200 chars) exists.
func ListChecklistFeedChannels(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]ChecklistChannel, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.name, c.sort_order,
       m.created_at AS latest_at,
       COALESCE(m.body, '') AS latest_title,
       w.body_len,
       w.posted_at,
       COALESCE(w.author_is_staff, false) AS author_is_staff
FROM course.feed_channels c
LEFT JOIN LATERAL (
	SELECT fm.created_at, LEFT(fm.body, 120) AS body
	FROM course.feed_messages fm
	WHERE fm.channel_id = c.id AND fm.parent_message_id IS NULL
	ORDER BY fm.created_at DESC
	LIMIT 1
) m ON TRUE
LEFT JOIN LATERAL (
	SELECT LENGTH(fm.body) AS body_len,
	       fm.created_at AS posted_at,
	       EXISTS (
	         SELECT 1 FROM course.course_enrollments ce
	         WHERE ce.course_id = c.course_id
	           AND ce.user_id = fm.author_user_id
	           AND ce.active
	           AND ce.role IN ('teacher', 'instructor', 'ta', 'designer', 'admin')
	       ) AS author_is_staff
	FROM course.feed_messages fm
	WHERE fm.channel_id = c.id
	  AND fm.parent_message_id IS NULL
	  AND lower(c.name) = 'announcements'
	  AND LENGTH(fm.body) >= 200
	  AND EXISTS (
	    SELECT 1 FROM course.course_enrollments ce
	    WHERE ce.course_id = c.course_id
	      AND ce.user_id = fm.author_user_id
	      AND ce.active
	      AND ce.role IN ('teacher', 'instructor', 'ta', 'designer', 'admin')
	  )
	ORDER BY fm.created_at ASC
	LIMIT 1
) w ON TRUE
WHERE c.course_id = $1 AND c.group_id IS NULL
ORDER BY c.sort_order ASC, c.created_at ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChecklistChannel{}
	for rows.Next() {
		var ch ChecklistChannel
		var latestAt *time.Time
		var title string
		var bodyLen *int
		var postedAt *time.Time
		var authorIsStaff bool
		if err := rows.Scan(
			&ch.ID, &ch.Name, &ch.SortOrder, &latestAt, &title,
			&bodyLen, &postedAt, &authorIsStaff,
		); err != nil {
			return nil, err
		}
		ch.LatestAt = latestAt
		ch.LatestTitle = title
		if bodyLen != nil && *bodyLen >= 200 && authorIsStaff {
			ch.StaffWelcome = &StaffWelcome{
				BodyLen:       *bodyLen,
				AuthorIsStaff: true,
				PostedAt:      postedAt,
			}
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}
