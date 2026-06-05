package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/authz"
	"github.com/lextures/lextures/server/internal/models/search"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrants"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

const searchQueryPerGroupLimit = 5

func (d Deps) handleSearchIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courses, err := course.ListForSearchIndex(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		peopleRaw, err := enrollment.ListPeopleForEnrolledCourses(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		grants, err := rbac.ListGrantedPermissionStrings(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		people := filterSearchPeopleByRosterRead(grants, peopleRaw)
		// JSON-encode nil slices as [] so clients can iterate without null checks.
		if courses == nil {
			courses = []search.CourseItem{}
		}
		if people == nil {
			people = []search.PersonItem{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(search.IndexResponse{
			Courses: courses,
			People:  people,
		})
	}
}

func (d Deps) handleSearchQuery() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) < 2 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Query must be at least 2 characters.")
			return
		}
		scope := strings.TrimSpace(r.URL.Query().Get("scope"))
		var scopePtr *string
		if scope != "" {
			scopePtr = &scope
		}
		types := parseSearchQueryTypes(r.URL.Query().Get("types"))
		start := time.Now()

		grants, err := rbac.ListGrantedPermissionStrings(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
			return
		}
		rosterCourses, rosterAll := courseCodesForPermission(grants, "enrollments:read")
		gradebookCourses, _ := courseCodesForPermission(grants, "gradebook:view")
		editCourses, editAll := courseCodesForPermissions(grants, "item:create", "items:create")

		var groups []search.QueryGroup
		ctx := r.Context()

		if types["course"] {
			items, total, err := course.SearchCoursesQuery(ctx, d.Pool, userID, q, scopePtr, searchQueryPerGroupLimit)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
				return
			}
			if items == nil {
				items = []search.QueryResultItem{}
			}
			groups = append(groups, search.QueryGroup{
				Type:  "course",
				Label: "Courses",
				Total: total,
				Items: items,
			})
		}

		if types["content"] {
			items, total, err := course.SearchContentQuery(
				ctx, d.Pool, userID, q, scopePtr, editCourses, editAll, searchQueryPerGroupLimit,
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
				return
			}
			if items == nil {
				items = []search.QueryResultItem{}
			}
			groups = append(groups, search.QueryGroup{
				Type:  "content",
				Label: "Content",
				Total: total,
				Items: items,
			})
		}

		if types["person"] && (rosterAll || len(rosterCourses) > 0) {
			rosterFilter := rosterCourses
			if rosterAll {
				rosterFilter = nil
			}
			items, total, err := enrollment.SearchPeopleQuery(
				ctx, d.Pool, userID, q, scopePtr, rosterFilter, gradebookCourses, searchQueryPerGroupLimit,
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Search failed.")
				return
			}
			if items == nil {
				items = []search.QueryResultItem{}
			}
			groups = append(groups, search.QueryGroup{
				Type:  "person",
				Label: "People",
				Total: total,
				Items: items,
			})
		}

		if groups == nil {
			groups = []search.QueryGroup{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(search.QueryResponse{
			Groups: groups,
			TookMs: time.Since(start).Milliseconds(),
		})
	}
}

func parseSearchQueryTypes(raw string) map[string]bool {
	defaultTypes := map[string]bool{
		"course":  true,
		"person":  true,
		"content": true,
	}
	if strings.TrimSpace(raw) == "" {
		return defaultTypes
	}
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "course":
			out["course"] = true
		case "person":
			out["person"] = true
		case "content":
			out["content"] = true
		}
	}
	if len(out) == 0 {
		return defaultTypes
	}
	return out
}

func courseCodesForPermission(grants []string, suffix string) (map[string]struct{}, bool) {
	return courseCodesForPermissions(grants, suffix)
}

func courseCodesForPermissions(grants []string, suffixes ...string) (map[string]struct{}, bool) {
	out := map[string]struct{}{}
	all := false
	for _, g := range grants {
		if !strings.HasPrefix(g, "course:") {
			continue
		}
		parts := strings.Split(g, ":")
		if len(parts) != 4 {
			continue
		}
		matched := false
		for _, suffix := range suffixes {
			if parts[2]+":"+parts[3] == suffix {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		code := parts[1]
		if code == "*" {
			all = true
			continue
		}
		if code == coursegrants.CourseCodePlaceholder {
			continue
		}
		out[code] = struct{}{}
	}
	return out, all
}

func filterSearchPeopleByRosterRead(grants []string, in []search.PersonItem) []search.PersonItem {
	if len(in) == 0 {
		return in
	}
	var out []search.PersonItem
	for _, p := range in {
		req := coursegrants.CourseEnrollmentsReadPermission(p.CourseCode)
		if authz.AnyGrantMatch(grants, req) {
			out = append(out, p)
		}
	}
	return out
}
