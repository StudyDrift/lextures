package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/service/adminsearch"
)

const adminSearchRateLimitPerMinute = 30

type adminSearchRateEntry struct {
	count int
	reset time.Time
}

var (
	adminSearchRateMu sync.Mutex
	adminSearchRates  = map[uuid.UUID]*adminSearchRateEntry{}
)

func (d Deps) adminSearchEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AdminSearchEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Admin search is not enabled.")
		return false
	}
	return true
}

func (d Deps) checkAdminSearchRateLimit(userID uuid.UUID) bool {
	adminSearchRateMu.Lock()
	defer adminSearchRateMu.Unlock()
	now := time.Now()
	e, ok := adminSearchRates[userID]
	if !ok || now.After(e.reset) {
		adminSearchRates[userID] = &adminSearchRateEntry{count: 1, reset: now.Add(time.Minute)}
		return true
	}
	if e.count >= adminSearchRateLimitPerMinute {
		return false
	}
	e.count++
	return true
}

func (d Deps) registerAdminSearchRoutes(r interface {
	Get(string, http.HandlerFunc)
}) {
	r.Get("/api/v1/admin/search", d.handleAdminSearchOmnisearch())
	r.Get("/api/v1/admin/search/users", d.handleAdminSearchUsers())
	r.Get("/api/v1/admin/search/courses", d.handleAdminSearchCourses())
	r.Get("/api/v1/admin/search/content", d.handleAdminSearchContent())
}

func (d Deps) handleAdminSearchOmnisearch() http.HandlerFunc {
	svc := adminsearch.New(d.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminSearchAccess(w, r)
		if !ok {
			return
		}
		if !d.checkAdminSearchRateLimit(actor) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Admin search rate limit exceeded.")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) < 2 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Query must be at least 2 characters.")
			return
		}
		types := adminsearch.ParseTypes(r.URL.Query().Get("types"))
		resp, err := svc.Omnisearch(r.Context(), orgID, q, types)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		svc.LogSearch(r.Context(), actor, orgID, q, len(resp.Users), len(resp.Courses), len(resp.Content), resp.TookMs)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handleAdminSearchUsers() http.HandlerFunc {
	svc := adminsearch.New(d.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminSearchAccess(w, r)
		if !ok {
			return
		}
		if !d.checkAdminSearchRateLimit(actor) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Admin search rate limit exceeded.")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) < 2 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Query must be at least 2 characters.")
			return
		}
		page, perPage := parseAdminSearchPagination(r)
		resp, err := svc.SearchUsersPaginated(r.Context(), orgID, q, page, perPage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handleAdminSearchCourses() http.HandlerFunc {
	svc := adminsearch.New(d.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminSearchAccess(w, r)
		if !ok {
			return
		}
		if !d.checkAdminSearchRateLimit(actor) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Admin search rate limit exceeded.")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) < 2 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Query must be at least 2 characters.")
			return
		}
		page, perPage := parseAdminSearchPagination(r)
		resp, err := svc.SearchCoursesPaginated(r.Context(), orgID, q, page, perPage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handleAdminSearchContent() http.HandlerFunc {
	svc := adminsearch.New(d.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminSearchAccess(w, r)
		if !ok {
			return
		}
		if !d.checkAdminSearchRateLimit(actor) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Admin search rate limit exceeded.")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) < 2 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Query must be at least 2 characters.")
			return
		}
		page, perPage := parseAdminSearchPagination(r)
		resp, err := svc.SearchContentPaginated(r.Context(), orgID, q, page, perPage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// adminSearchAccess authenticates and ensures org_admin or global admin access.
func (d Deps) adminSearchAccess(w http.ResponseWriter, r *http.Request) (actor uuid.UUID, targetOrg uuid.UUID, globalAdmin bool, ok bool) {
	if !d.adminSearchEnabled(w) {
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	return d.adminConsoleAccess(w, r, false)
}

func parseAdminSearchPagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 25
	if s := strings.TrimSpace(r.URL.Query().Get("page")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			page = n
		}
	}
	if s := strings.TrimSpace(r.URL.Query().Get("per_page")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			perPage = n
		}
	}
	return page, perPage
}
