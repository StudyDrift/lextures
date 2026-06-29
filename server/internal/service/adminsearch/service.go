// Package adminsearch implements org-wide admin search (plan 18.4).
package adminsearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"expvar"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/adminsearch"
	adminsearchrepo "github.com/lextures/lextures/server/internal/repos/adminsearch"
)

const omnisearchPerTypeLimit = 5

var (
	searchesTotal   atomic.Uint64
	resultsTotal    expvar.Map
	durationMsTotal atomic.Uint64
	durationCount   atomic.Uint64
)

func init() {
	expvar.Publish("admin_search_searches_total", expvar.Func(func() any { return searchesTotal.Load() }))
	expvar.Publish("admin_search_results_total", &resultsTotal)
	expvar.Publish("admin_search_duration_ms_avg", expvar.Func(func() any {
		n := durationCount.Load()
		if n == 0 {
			return float64(0)
		}
		return float64(durationMsTotal.Load()) / float64(n)
	}))
}

var emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

// Service exposes org-scoped admin search operations.
type Service struct {
	pool *pgxpool.Pool
}

// New constructs an admin search service backed by the given pool.
func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// TypesFilter controls which entity types are included in omnisearch.
type TypesFilter struct {
	Users   bool
	Courses bool
	Content bool
}

// ParseTypes parses the comma-separated types query parameter.
func ParseTypes(raw string) TypesFilter {
	all := TypesFilter{Users: true, Courses: true, Content: true}
	if strings.TrimSpace(raw) == "" {
		return all
	}
	out := TypesFilter{}
	for _, part := range strings.Split(raw, ",") {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "users", "user":
			out.Users = true
		case "courses", "course":
			out.Courses = true
		case "content":
			out.Content = true
		}
	}
	if !out.Users && !out.Courses && !out.Content {
		return all
	}
	return out
}

// Omnisearch returns up to 5 results per entity type for the given org.
func (s *Service) Omnisearch(
	ctx context.Context,
	orgID uuid.UUID,
	q string,
	types TypesFilter,
) (adminsearch.OmnisearchResponse, error) {
	start := time.Now()
	searchesTotal.Add(1)

	var resp adminsearch.OmnisearchResponse
	if types.Users {
		items, _, err := adminsearchrepo.SearchUsers(ctx, s.pool, orgID, q, omnisearchPerTypeLimit, 0)
		if err != nil {
			return resp, err
		}
		if items == nil {
			items = []adminsearch.Result{}
		}
		resp.Users = items
		recordResults("users", len(items))
	} else {
		resp.Users = []adminsearch.Result{}
	}

	if types.Courses {
		items, _, err := adminsearchrepo.SearchCourses(ctx, s.pool, orgID, q, omnisearchPerTypeLimit, 0)
		if err != nil {
			return resp, err
		}
		if items == nil {
			items = []adminsearch.Result{}
		}
		resp.Courses = items
		recordResults("courses", len(items))
	} else {
		resp.Courses = []adminsearch.Result{}
	}

	if types.Content {
		items, _, err := adminsearchrepo.SearchContent(ctx, s.pool, orgID, q, omnisearchPerTypeLimit, 0)
		if err != nil {
			return resp, err
		}
		if items == nil {
			items = []adminsearch.Result{}
		}
		resp.Content = items
		recordResults("content", len(items))
	} else {
		resp.Content = []adminsearch.Result{}
	}

	resp.TookMs = time.Since(start).Milliseconds()
	recordDuration(resp.TookMs)
	return resp, nil
}

// SearchUsersPaginated returns a paginated page of user results.
func (s *Service) SearchUsersPaginated(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) (adminsearch.PaginatedUsers, error) {
	start := time.Now()
	page, perPage = normalizePagination(page, perPage)
	offset := (page - 1) * perPage
	items, total, err := adminsearchrepo.SearchUsers(ctx, s.pool, orgID, q, perPage, offset)
	if err != nil {
		return adminsearch.PaginatedUsers{}, err
	}
	if items == nil {
		items = []adminsearch.Result{}
	}
	took := time.Since(start).Milliseconds()
	recordDuration(took)
	return adminsearch.PaginatedUsers{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages(total, perPage),
		TookMs:     took,
	}, nil
}

// SearchCoursesPaginated returns a paginated page of course results.
func (s *Service) SearchCoursesPaginated(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) (adminsearch.PaginatedCourses, error) {
	start := time.Now()
	page, perPage = normalizePagination(page, perPage)
	offset := (page - 1) * perPage
	items, total, err := adminsearchrepo.SearchCourses(ctx, s.pool, orgID, q, perPage, offset)
	if err != nil {
		return adminsearch.PaginatedCourses{}, err
	}
	if items == nil {
		items = []adminsearch.Result{}
	}
	took := time.Since(start).Milliseconds()
	recordDuration(took)
	return adminsearch.PaginatedCourses{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages(total, perPage),
		TookMs:     took,
	}, nil
}

// SearchContentPaginated returns a paginated page of content results.
func (s *Service) SearchContentPaginated(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) (adminsearch.PaginatedContent, error) {
	start := time.Now()
	page, perPage = normalizePagination(page, perPage)
	offset := (page - 1) * perPage
	items, total, err := adminsearchrepo.SearchContent(ctx, s.pool, orgID, q, perPage, offset)
	if err != nil {
		return adminsearch.PaginatedContent{}, err
	}
	if items == nil {
		items = []adminsearch.Result{}
	}
	took := time.Since(start).Milliseconds()
	recordDuration(took)
	return adminsearch.PaginatedContent{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages(total, perPage),
		TookMs:     took,
	}, nil
}

// LogSearch records a scrubbed search query for analytics (best-effort).
func (s *Service) LogSearch(
	ctx context.Context,
	actorID, orgID uuid.UUID,
	rawQuery string,
	userCount, courseCount, contentCount int,
	tookMs int64,
) {
	scrubbed := ScrubQueryPII(rawQuery)
	_ = adminsearchrepo.InsertSearchLog(ctx, s.pool, actorID, orgID, scrubbed, userCount, courseCount, contentCount, tookMs)
}

// ScrubQueryPII removes or hashes email addresses before persisting search queries.
func ScrubQueryPII(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	if !emailPattern.MatchString(q) {
		return q
	}
	return emailPattern.ReplaceAllStringFunc(q, func(email string) string {
		sum := sha256.Sum256([]byte(strings.ToLower(email)))
		return "email:" + hex.EncodeToString(sum[:8])
	})
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch perPage {
	case 50, 100:
	default:
		perPage = 25
	}
	return page, perPage
}

func totalPages(total int64, perPage int) int {
	if perPage <= 0 || total <= 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}

func recordResults(entityType string, count int) {
	if count > 0 {
		resultsTotal.Add(entityType, int64(count))
	}
}

func recordDuration(ms int64) {
	durationMsTotal.Add(uint64(ms))
	durationCount.Add(1)
}
