package oersearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/oercache"
	"github.com/lextures/lextures/server/internal/repos/oerproviders"
)

const cacheTTL = 24 * time.Hour

// Service proxies OER catalog searches with Postgres caching (plan 8.9).
type Service struct {
	Pool    *pgxpool.Pool
	Stub    bool
	commons Provider
	merlot  Provider
	openstax Provider
}

// New builds the OER search service. When stub is true, all providers use local sample data.
func New(pool *pgxpool.Pool, stub bool) *Service {
	s := &Service{Pool: pool, Stub: stub}
	if stub {
		s.commons = newStubProvider("oer_commons")
		s.merlot = newStubProvider("merlot")
		s.openstax = newStubProvider("openstax")
	} else {
		s.commons = newStubProvider("oer_commons")
		s.merlot = newStubProvider("merlot")
		s.openstax = newOpenStaxProvider()
	}
	return s
}

func (s *Service) providerByID(id string) (Provider, bool) {
	switch id {
	case "oer_commons":
		return s.commons, true
	case "merlot":
		return s.merlot, true
	case "openstax":
		return s.openstax, true
	default:
		return nil, false
	}
}

func queryHash(params SearchParams) string {
	payload, _ := json.Marshal(struct {
		Q       string `json:"q"`
		Subject string `json:"subject"`
		Level   string `json:"level"`
		License string `json:"license"`
	}{
		Q: strings.TrimSpace(params.Query),
		Subject: strings.TrimSpace(params.Subject),
		Level: strings.TrimSpace(params.Level),
		License: strings.TrimSpace(params.License),
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Search runs a cached search for one provider.
func (s *Service) Search(ctx context.Context, providerID string, params SearchParams) (SearchResponse, error) {
	if s.Pool == nil {
		return SearchResponse{}, errors.New("db pool is nil")
	}
	enabled, err := oerproviders.IsEnabled(ctx, s.Pool, providerID)
	if err != nil {
		return SearchResponse{}, err
	}
	if !enabled {
		return SearchResponse{}, errors.New("oer provider disabled")
	}
	p, ok := s.providerByID(providerID)
	if !ok {
		return SearchResponse{}, errors.New("unknown oer provider")
	}
	now := time.Now().UTC()
	hash := queryHash(params)
	cached, err := oercache.Get(ctx, s.Pool, providerID, hash, now)
	if err != nil {
		return SearchResponse{}, err
	}
	if cached != nil {
		var results []Result
		if err := json.Unmarshal(cached.Results, &results); err == nil {
			slog.Info("oer_search_requests_total", "provider", providerID, "query", params.Query, "result_count", len(results), "cache", "hit")
			return SearchResponse{
				Results:   results,
				Provider:  providerID,
				FromCache: true,
				CacheAsOf: cached.FetchedAt.UTC().Format(time.RFC3339),
			}, nil
		}
	}

	results, fetchErr := p.Search(ctx, params)
	if fetchErr != nil {
		staleRow, serr := oercache.GetAny(ctx, s.Pool, providerID, hash)
		if serr == nil && staleRow != nil {
			var staleResults []Result
			if err := json.Unmarshal(staleRow.Results, &staleResults); err == nil {
				slog.Warn("oer search provider unavailable, serving stale cache", "provider", providerID, "err", fetchErr)
				return SearchResponse{
					Results:    staleResults,
					Provider:   providerID,
					FromCache:  true,
					CacheAsOf:  staleRow.FetchedAt.UTC().Format(time.RFC3339),
					StaleCache: true,
				}, nil
			}
		}
		return SearchResponse{}, fetchErr
	}

	raw, err := json.Marshal(results)
	if err != nil {
		return SearchResponse{}, err
	}
	expires := now.Add(cacheTTL)
	if err := oercache.Put(ctx, s.Pool, providerID, hash, raw, now, expires); err != nil {
		slog.Warn("oer cache put failed", "provider", providerID, "err", err)
	}
	slog.Info("oer_search_requests_total", "provider", providerID, "query", params.Query, "result_count", len(results), "cache", "miss")
	return SearchResponse{
		Results:  results,
		Provider: providerID,
		FromCache: false,
	}, nil
}

// EnabledProviderIDs lists providers turned on in settings.
func (s *Service) EnabledProviderIDs(ctx context.Context) ([]string, error) {
	if s.Pool == nil {
		return nil, errors.New("db pool is nil")
	}
	return oerproviders.EnabledProviders(ctx, s.Pool)
}
