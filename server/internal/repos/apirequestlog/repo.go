// Package apirequestlog persists authenticated public API audit entries (plan 16.1).
package apirequestlog

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Insert writes a request log row.
func Insert(ctx context.Context, pool *pgxpool.Pool, tokenID, userID *uuid.UUID, method, path string, status, latencyMs int, ip, pepper string) error {
	if pool == nil {
		return nil
	}
	var ipHash *string
	ip = strings.TrimSpace(ip)
	if ip != "" {
		h := hmac.New(sha256.New, []byte(pepper))
		_, _ = h.Write([]byte(ip))
		s := hex.EncodeToString(h.Sum(nil))
		ipHash = &s
	}
	_, err := pool.Exec(ctx, `
INSERT INTO api.request_log (token_id, user_id, method, path, status, latency_ms, ip_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, tokenID, userID, method, path, status, latencyMs, ipHash)
	return err
}

// LogAsync schedules a log insert without blocking the response path.
func LogAsync(pool *pgxpool.Pool, tokenID, userID *uuid.UUID, method, path string, status, latencyMs int, ip, pepper string) {
	if pool == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = Insert(ctx, pool, tokenID, userID, method, path, status, latencyMs, ip, pepper)
	}()
}
