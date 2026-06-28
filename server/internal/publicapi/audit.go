package publicapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogRequest inserts an audit row for an authenticated public API call.
func LogRequest(ctx context.Context, pool *pgxpool.Pool, tokenID, userID *uuid.UUID, method, path string, status, latencyMs int, remoteIP string) {
	if pool == nil {
		return
	}
	var ipHash *string
	if remoteIP != "" {
		host, _, err := net.SplitHostPort(remoteIP)
		if err != nil {
			host = remoteIP
		}
		sum := sha256.Sum256([]byte(host))
		h := hex.EncodeToString(sum[:])
		ipHash = &h
	}
	_, _ = pool.Exec(ctx, `
INSERT INTO api.request_log (token_id, user_id, method, path, status, latency_ms, ip_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, tokenID, userID, method, path, status, latencyMs, ipHash)
}

// ElapsedMs returns milliseconds since start.
func ElapsedMs(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}

// RequestLogRetention is the GDPR retention window for API request logs
// (plan 17.4 NFR privacy). Rows older than this are deleted by the
// request_log_retention scheduled job.
const RequestLogRetention = 90 * 24 * time.Hour

// DeleteRequestLogsOlderThan removes api.request_log rows created before cutoff,
// returning the number deleted (plan 17.4 FR-4, AC-6).
func DeleteRequestLogsOlderThan(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM api.request_log WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
