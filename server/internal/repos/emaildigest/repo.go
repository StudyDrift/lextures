package emaildigest

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Item is one line in a daily digest.
type Item struct {
	EventType   string
	SummaryLine string
	DetailURL   string
}

// Candidate is a user with pending digest items and their timezone.
type Candidate struct {
	UserID   uuid.UUID
	Timezone *string
}

// Append adds a digest line for a user.
func Append(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, eventType, summaryLine, detailURL string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO settings.email_digest_items (user_id, event_type, summary_line, detail_url)
VALUES ($1, $2, $3, NULLIF($4, ''))
`, userID, eventType, summaryLine, detailURL)
	return err
}

// ListAndClear returns all digest items for a user since the given time and deletes them.
func ListAndClear(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, since time.Time) ([]Item, error) {
	rows, err := pool.Query(ctx, `
DELETE FROM settings.email_digest_items
WHERE user_id = $1 AND created_at >= $2
RETURNING event_type, summary_line, COALESCE(detail_url, '')
`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.EventType, &it.SummaryLine, &it.DetailURL); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListCandidates returns users with pending digest items and their timezone.
func ListCandidates(ctx context.Context, pool *pgxpool.Pool) ([]Candidate, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT d.user_id, u.timezone
FROM settings.email_digest_items d
JOIN "user".users u ON u.id = d.user_id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.Timezone); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
