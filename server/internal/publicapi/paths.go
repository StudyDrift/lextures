package publicapi

import (
	"net/http"
	"strings"
)

// Route describes a public API endpoint.
type Route struct {
	Method      string
	PathPattern string // literal path prefix or exact path
	Scope       string // empty = public (no auth)
	UUIDSegment int    // 1-based index of {id} UUID segment, 0 = none
	Subpath     string // suffix after UUID, e.g. "/enrollments"
}

// Routes is the public API surface (plan 16.1 §9).
var Routes = []Route{
	{Method: http.MethodGet, PathPattern: "/api/v1/openapi.json"},
	{Method: http.MethodGet, PathPattern: "/api/v1/docs"},
	{Method: http.MethodGet, PathPattern: "/api/v1/redoc"},
	{Method: http.MethodGet, PathPattern: "/api/v1/courses", Scope: "courses:read"},
	{Method: http.MethodGet, PathPattern: "/api/v1/courses", Scope: "courses:read", UUIDSegment: 1},
	{Method: http.MethodGet, PathPattern: "/api/v1/courses", Scope: "enrollments:read", UUIDSegment: 1, Subpath: "/enrollments"},
	{Method: http.MethodPost, PathPattern: "/api/v1/courses", Scope: "enrollments:write", UUIDSegment: 1, Subpath: "/enrollments"},
	{Method: http.MethodPost, PathPattern: "/api/v1/courses", Scope: "courses:write", UUIDSegment: 1, Subpath: "/announcements"},
	{Method: http.MethodGet, PathPattern: "/api/v1/users", Scope: "users:read", UUIDSegment: 1},
	{Method: http.MethodGet, PathPattern: "/api/v1/assignments", Scope: "assignments:read"},
	{Method: http.MethodGet, PathPattern: "/api/v1/grades", Scope: "grades:read"},
	{Method: http.MethodPost, PathPattern: "/api/v1/grades", Scope: "grades:write"},
	{Method: http.MethodGet, PathPattern: "/api/v1/graphql", Scope: "graphql:read"},
}

// Match returns the matched route and extracted UUID id (if any).
func Match(method, path string) (matched Route, id string, ok bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSuffix(strings.TrimSpace(path), "/")
	if path == "" {
		path = "/"
	}
	for _, rt := range Routes {
		if rt.Method != method {
			continue
		}
		base := strings.TrimSuffix(rt.PathPattern, "/")
		if rt.UUIDSegment == 0 && rt.Subpath == "" {
			if path == base {
				return rt, "", true
			}
			continue
		}
		prefix := base + "/"
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		if rt.Subpath != "" {
			idx := strings.Index(rest, "/")
			if idx < 0 {
				continue
			}
			idPart := rest[:idx]
			suffix := rest[idx:]
			if suffix != rt.Subpath || !isUUID(idPart) {
				continue
			}
			return rt, idPart, true
		}
		if strings.Contains(rest, "/") {
			continue
		}
		if !isUUID(rest) {
			continue
		}
		return rt, rest, true
	}
	return Route{}, "", false
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}
	}
	return true
}

// IsAccessKeyRequest is true when Authorization carries an ltk_ personal access key.
func IsAccessKeyRequest(r *http.Request) bool {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(h[7:]), "ltk_")
}
