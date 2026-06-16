// Package catalogsearch implements the public course catalog search and indexing
// service (plan 15.1). It wraps the Postgres full-text catalog queries behind a
// small service surface so the search backend can be swapped (e.g. for OpenSearch
// or Algolia) without changing HTTP handlers.
package catalogsearch

import (
	"context"
	"expvar"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

var (
	searchesTotal  atomic.Uint64
	pageViewsTotal atomic.Uint64
)

func init() {
	expvar.Publish("catalog_searches_total", expvar.Func(func() any { return searchesTotal.Load() }))
	expvar.Publish("catalog_page_views_total", expvar.Func(func() any { return pageViewsTotal.Load() }))
}

// Service exposes the public catalog search operations.
type Service struct {
	pool *pgxpool.Pool
}

// New constructs a catalog search service backed by the given pool.
func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// SearchResult is a page of catalog courses plus pagination metadata.
type SearchResult struct {
	Courses    []repoCourse.PublicCatalogCourse `json:"courses"`
	Total      int                              `json:"total"`
	NextCursor string                           `json:"nextCursor"`
}

// Search returns a page of public catalog courses for the given filter and records
// the catalog_searches_total metric.
func (s *Service) Search(ctx context.Context, f repoCourse.PublicCatalogFilter) (SearchResult, error) {
	searchesTotal.Add(1)
	courses, total, next, err := repoCourse.ListPublicCatalog(ctx, s.pool, f)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Courses: courses, Total: total, NextCursor: next}, nil
}

// CourseBySlug returns a single public course landing record (or nil) and records
// the catalog_page_views_total metric.
func (s *Service) CourseBySlug(ctx context.Context, slug string) (*repoCourse.PublicCatalogCourse, error) {
	c, err := repoCourse.GetPublicCourseBySlug(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	if c != nil {
		pageViewsTotal.Add(1)
	}
	return c, nil
}

// Categories returns the catalog browse taxonomy derived from published courses.
func (s *Service) Categories(ctx context.Context) ([]repoCourse.CatalogCategory, error) {
	return repoCourse.ListCatalogCategories(ctx, s.pool)
}
