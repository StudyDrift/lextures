// Package adminsearch holds JSON types for org-wide admin search (plan 18.4).
package adminsearch

// Result is one search hit across users, courses, or content.
type Result struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Snippet  string  `json:"snippet,omitempty"`
	Path     string  `json:"path"`
	Score    float64 `json:"score,omitempty"`
}

// OmnisearchResponse is the top-level GET /api/v1/admin/search payload.
type OmnisearchResponse struct {
	Users   []Result `json:"users"`
	Courses []Result `json:"courses"`
	Content []Result `json:"content"`
	TookMs  int64    `json:"tookMs"`
}

// PaginatedUsers is a page of user search results.
type PaginatedUsers struct {
	Items      []Result `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"perPage"`
	TotalPages int      `json:"totalPages"`
	TookMs     int64    `json:"tookMs"`
}

// PaginatedCourses is a page of course search results.
type PaginatedCourses struct {
	Items      []Result `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"perPage"`
	TotalPages int      `json:"totalPages"`
	TookMs     int64    `json:"tookMs"`
}

// PaginatedContent is a page of content search results.
type PaginatedContent struct {
	Items      []Result `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"perPage"`
	TotalPages int      `json:"totalPages"`
	TookMs     int64    `json:"tookMs"`
}
