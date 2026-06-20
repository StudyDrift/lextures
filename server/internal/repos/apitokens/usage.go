package apitokens

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageEvent is a deferred last-used update for a token (plan 16.2 FR-6).
type UsageEvent struct {
	TokenID uuid.UUID
	IPHash  string
	At      time.Time
}

var (
	usageMu    sync.Mutex
	usageBatch = make(map[uuid.UUID]UsageEvent)
)

// RecordUsage queues an async last-used update; drops on overload.
func RecordUsage(tokenID uuid.UUID, ipHash string) {
	usageMu.Lock()
	defer usageMu.Unlock()
	usageBatch[tokenID] = UsageEvent{
		TokenID: tokenID,
		IPHash:  ipHash,
		At:      time.Now().UTC(),
	}
}

// FlushUsage writes queued usage events to the database.
func FlushUsage(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	if pool == nil {
		return 0, nil
	}
	usageMu.Lock()
	if len(usageBatch) == 0 {
		usageMu.Unlock()
		return 0, nil
	}
	batch := usageBatch
	usageBatch = make(map[uuid.UUID]UsageEvent)
	usageMu.Unlock()

	n := 0
	for _, ev := range batch {
		var err error
		if ev.IPHash != "" {
			_, err = pool.Exec(ctx, `
UPDATE auth.api_tokens
SET last_used_at = $2, last_used_ip_hash = $3
WHERE id = $1
`, ev.TokenID, ev.At, ev.IPHash)
		} else {
			_, err = pool.Exec(ctx, `
UPDATE auth.api_tokens SET last_used_at = $2 WHERE id = $1
`, ev.TokenID, ev.At)
		}
		if err == nil {
			n++
		}
	}
	return n, nil
}

// ResetUsageQueue clears pending usage events (tests only).
func ResetUsageQueue() {
	usageMu.Lock()
	defer usageMu.Unlock()
	usageBatch = make(map[uuid.UUID]UsageEvent)
}
