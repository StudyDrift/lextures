// Package oercache stores cached OER provider search responses (plan 8.9).
package oercache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CachedResults is the JSON payload stored per provider/query hash.
type CachedResults struct {
	Results   json.RawMessage `json:"results"`
	FetchedAt time.Time       `json:"fetchedAt"`
}

// GetAny returns the most recent cache row regardless of expiry (for stale fallback).
func GetAny(ctx context.Context, pool *pgxpool.Pool, provider, queryHash string) (*CachedResults, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var raw []byte
	var fetchedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT results_json, fetched_at
FROM content.oer_search_cache
WHERE provider = $1 AND query_hash = $2
`, provider, queryHash).Scan(&raw, &fetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &CachedResults{Results: raw, FetchedAt: fetchedAt}, nil
}

// Get returns non-expired cached results, or nil if missing/expired.
func Get(ctx context.Context, pool *pgxpool.Pool, provider, queryHash string, now time.Time) (*CachedResults, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var raw []byte
	var fetchedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT results_json, fetched_at
FROM content.oer_search_cache
WHERE provider = $1 AND query_hash = $2 AND expires_at > $3
`, provider, queryHash, now).Scan(&raw, &fetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &CachedResults{Results: raw, FetchedAt: fetchedAt}, nil
}

// Put upserts cached search results with a 24-hour TTL.
func Put(ctx context.Context, pool *pgxpool.Pool, provider, queryHash string, resultsJSON []byte, fetchedAt, expiresAt time.Time) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO content.oer_search_cache (provider, query_hash, results_json, fetched_at, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider, query_hash) DO UPDATE SET
    results_json = EXCLUDED.results_json,
    fetched_at = EXCLUDED.fetched_at,
    expires_at = EXCLUDED.expires_at
`, provider, queryHash, resultsJSON, fetchedAt, expiresAt)
	return err
}

// DeleteExpired removes cache rows past their expiry (optional housekeeping).
func DeleteExpired(ctx context.Context, pool *pgxpool.Pool, before time.Time) (int64, error) {
	if pool == nil {
		return 0, errors.New("db pool is nil")
	}
	tag, err := pool.Exec(ctx, `DELETE FROM content.oer_search_cache WHERE expires_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
