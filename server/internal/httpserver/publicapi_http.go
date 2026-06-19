package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/publicapi"
)

// publicAPIMiddleware intercepts ltk_ bearer requests to the versioned public API surface (plan 16.1).
// JWT session requests pass through to the existing SPA handlers unchanged.
func (d Deps) publicAPIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt, id, matched := publicapi.Match(r.Method, r.URL.Path)
		if !matched {
			next.ServeHTTP(w, r)
			return
		}
		// Meta endpoints are always reachable; docs require ENABLE_API_DOCS.
		if rt.PathPattern == "/api/v1/openapi.json" {
			publicapi.ServeOpenAPI(w, r)
			return
		}
		if rt.PathPattern == "/api/v1/docs" {
			if !d.effectiveConfig().EnableAPIDocs {
				publicapi.WriteNotFound(w, r.URL.Path)
				return
			}
			publicapi.ServeSwaggerUI(w, r)
			return
		}
		if rt.PathPattern == "/api/v1/redoc" {
			if !d.effectiveConfig().EnableAPIDocs {
				publicapi.WriteNotFound(w, r.URL.Path)
				return
			}
			publicapi.ServeReDoc(w, r)
			return
		}
		// Protected resources: JWT/session requests use the legacy SPA handlers unchanged.
		if !publicapi.IsAccessKeyRequest(r) {
			if d.effectiveConfig().FFPublicAPI && rt.Scope != "" && strings.TrimSpace(r.Header.Get("Authorization")) == "" {
				publicapi.WriteUnauthorized(w, r.URL.Path)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if !d.effectiveConfig().FFPublicAPI {
			publicapi.WriteServiceUnavailable(w, r.URL.Path)
			return
		}
		start := time.Now()
		status, userID := d.servePublicAPI(w, r, rt, id)
		d.logPublicAPIRequest(r, userID, start, status)
	})
}

func (d Deps) servePublicAPI(w http.ResponseWriter, r *http.Request, rt publicapi.Route, id string) (int, uuid.UUID) {
	userID, ctx, tok, ok := d.publicAPIAuth(w, r, rt.Scope)
	if !ok {
		return http.StatusUnauthorized, uuid.UUID{}
	}
	*r = *r.WithContext(ctx)

	if tok != nil {
		key := tok.TokenID.String()
		if allowed, retry := publicapi.DefaultLimiter.Allow(key, time.Now()); !allowed {
			publicapi.WriteRateLimited(w, r.URL.Path, retry)
			return http.StatusTooManyRequests, userID
		}
	}

	switch {
	case rt.PathPattern == "/api/v1/courses" && rt.Subpath == "" && id == "":
		return d.publicAPIListCourses(w, r, userID, tok), userID
	case rt.PathPattern == "/api/v1/courses" && rt.Subpath == "" && id != "":
		return d.publicAPIGetCourse(w, r, userID, id, tok), userID
	case rt.Subpath == "/enrollments":
		return d.publicAPIListEnrollments(w, r, userID, id, tok), userID
	case rt.PathPattern == "/api/v1/users":
		return d.publicAPIGetUser(w, r, userID, id, tok), userID
	case rt.PathPattern == "/api/v1/assignments":
		return d.publicAPIListAssignments(w, r, userID, tok), userID
	case rt.PathPattern == "/api/v1/grades":
		return d.publicAPIListGrades(w, r, userID, tok), userID
	case rt.PathPattern == "/api/v1/graphql":
		publicapi.ServeGraphQL(w, r)
		return http.StatusOK, userID
	default:
		publicapi.WriteNotFound(w, r.URL.Path)
		return http.StatusNotFound, userID
	}
}

func (d Deps) publicAPIAuth(w http.ResponseWriter, r *http.Request, requiredScope string) (uuid.UUID, context.Context, *auth.APITokenAuth, bool) {
	if d.JWTSigner == nil || d.Pool == nil {
		publicapi.WriteUnauthorized(w, r.URL.Path)
		return uuid.UUID{}, r.Context(), nil, false
	}
	u, ctx, err := auth.UserFromRequestOrAccessKey(r, d.JWTSigner, d.Pool)
	if err != nil || !publicapi.IsAccessKeyRequest(r) {
		publicapi.WriteUnauthorized(w, r.URL.Path)
		return uuid.UUID{}, r.Context(), nil, false
	}
	tok, _ := auth.APITokenFromContext(ctx)
	if tok == nil {
		publicapi.WriteUnauthorized(w, r.URL.Path)
		return uuid.UUID{}, ctx, nil, false
	}
	if requiredScope != "" && !publicapi.HasScope(tok.Scopes, requiredScope) {
		publicapi.WriteForbidden(w, r.URL.Path, "Token lacks required scope: "+requiredScope)
		return uuid.UUID{}, ctx, tok, false
	}
	userID, err := uuid.Parse(u.UserID)
	if err != nil {
		publicapi.WriteUnauthorized(w, r.URL.Path)
		return uuid.UUID{}, ctx, tok, false
	}
	return userID, ctx, tok, true
}

func (d Deps) publicAPIListCourses(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tok *auth.APITokenAuth) int {
	page, err := publicapi.ParsePageParams(r.URL.Query())
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, err.Error())
		return http.StatusBadRequest
	}
	var allowed []uuid.UUID
	if tok != nil {
		allowed = tok.CourseIDs
	}
	items, err := publicapi.ListCourses(r.Context(), d.Pool, userID, allowed)
	if err != nil {
		publicapi.WriteInternal(w, r.URL.Path)
		return http.StatusInternalServerError
	}
	slice, total := publicapi.SlicePage(items, page.Offset, page.Limit)
	fieldsets := publicapi.ParseFieldsets(r.URL.Query())
	data := publicapi.ApplyFieldsetsToCollection(slice, "courses", fieldsets)
	resp := publicapi.BuildCollectionResponse(data, total, page.Offset, page.Limit, r.URL.Path, r.URL.Query())
	publicapi.SetLinkHeader(w, resp.Links)
	writeJSON(w, http.StatusOK, resp)
	return http.StatusOK
}

func (d Deps) publicAPIGetCourse(w http.ResponseWriter, r *http.Request, userID uuid.UUID, id string, tok *auth.APITokenAuth) int {
	cid, err := uuid.Parse(id)
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, "Invalid course id.")
		return http.StatusBadRequest
	}
	if tok != nil && len(tok.CourseIDs) > 0 && !auth.AccessKeyAllowsCourse(r.Context(), cid) {
		publicapi.WriteNotFound(w, r.URL.Path)
		return http.StatusNotFound
	}
	c, err := publicapi.GetCourseByID(r.Context(), d.Pool, userID, cid)
	if err != nil {
		publicapi.WriteInternal(w, r.URL.Path)
		return http.StatusInternalServerError
	}
	if c == nil {
		publicapi.WriteNotFound(w, r.URL.Path)
		return http.StatusNotFound
	}
	writeJSON(w, http.StatusOK, c)
	return http.StatusOK
}

func (d Deps) publicAPIListEnrollments(w http.ResponseWriter, r *http.Request, userID uuid.UUID, id string, tok *auth.APITokenAuth) int {
	cid, err := uuid.Parse(id)
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, "Invalid course id.")
		return http.StatusBadRequest
	}
	if tok != nil && len(tok.CourseIDs) > 0 && !auth.AccessKeyAllowsCourse(r.Context(), cid) {
		publicapi.WriteNotFound(w, r.URL.Path)
		return http.StatusNotFound
	}
	page, err := publicapi.ParsePageParams(r.URL.Query())
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, err.Error())
		return http.StatusBadRequest
	}
	items, err := publicapi.ListEnrollments(r.Context(), d.Pool, userID, cid)
	if err != nil {
		publicapi.WriteInternal(w, r.URL.Path)
		return http.StatusInternalServerError
	}
	slice, total := publicapi.SlicePage(items, page.Offset, page.Limit)
	data := publicapi.ToAnySlice(slice)
	resp := publicapi.BuildCollectionResponse(data, total, page.Offset, page.Limit, r.URL.Path, r.URL.Query())
	publicapi.SetLinkHeader(w, resp.Links)
	writeJSON(w, http.StatusOK, resp)
	return http.StatusOK
}

func (d Deps) publicAPIGetUser(w http.ResponseWriter, r *http.Request, userID uuid.UUID, id string, tok *auth.APITokenAuth) int {
	uid, err := uuid.Parse(id)
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, "Invalid user id.")
		return http.StatusBadRequest
	}
	includePII := tok != nil && publicapi.HasScope(tok.Scopes, "pii:read")
	u, err := publicapi.GetUser(r.Context(), d.Pool, userID, uid, includePII)
	if err != nil {
		publicapi.WriteInternal(w, r.URL.Path)
		return http.StatusInternalServerError
	}
	if u == nil {
		publicapi.WriteNotFound(w, r.URL.Path)
		return http.StatusNotFound
	}
	writeJSON(w, http.StatusOK, u)
	return http.StatusOK
}

func (d Deps) publicAPIListAssignments(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tok *auth.APITokenAuth) int {
	page, err := publicapi.ParsePageParams(r.URL.Query())
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, err.Error())
		return http.StatusBadRequest
	}
	var allowed []uuid.UUID
	if tok != nil {
		allowed = tok.CourseIDs
	}
	items, err := publicapi.ListAssignments(r.Context(), d.Pool, userID, allowed)
	if err != nil {
		publicapi.WriteInternal(w, r.URL.Path)
		return http.StatusInternalServerError
	}
	slice, total := publicapi.SlicePage(items, page.Offset, page.Limit)
	data := publicapi.ToAnySlice(slice)
	resp := publicapi.BuildCollectionResponse(data, total, page.Offset, page.Limit, r.URL.Path, r.URL.Query())
	publicapi.SetLinkHeader(w, resp.Links)
	writeJSON(w, http.StatusOK, resp)
	return http.StatusOK
}

func (d Deps) publicAPIListGrades(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tok *auth.APITokenAuth) int {
	page, err := publicapi.ParsePageParams(r.URL.Query())
	if err != nil {
		publicapi.WriteBadRequest(w, r.URL.Path, err.Error())
		return http.StatusBadRequest
	}
	var allowed []uuid.UUID
	if tok != nil {
		allowed = tok.CourseIDs
	}
	items, err := publicapi.ListGrades(r.Context(), d.Pool, userID, allowed)
	if err != nil {
		publicapi.WriteInternal(w, r.URL.Path)
		return http.StatusInternalServerError
	}
	slice, total := publicapi.SlicePage(items, page.Offset, page.Limit)
	data := publicapi.ToAnySlice(slice)
	resp := publicapi.BuildCollectionResponse(data, total, page.Offset, page.Limit, r.URL.Path, r.URL.Query())
	publicapi.SetLinkHeader(w, resp.Links)
	writeJSON(w, http.StatusOK, resp)
	return http.StatusOK
}

func (d Deps) logPublicAPIRequest(r *http.Request, userID uuid.UUID, start time.Time, status int) {
	if d.Pool == nil {
		return
	}
	ctx := r.Context()
	tok, ok := auth.APITokenFromContext(ctx)
	if !ok || tok == nil {
		return
	}
	var uid *uuid.UUID
	if userID != (uuid.UUID{}) {
		uid = &userID
	}
	tid := tok.TokenID
	publicapi.LogRequest(ctx, d.Pool, &tid, uid, r.Method, r.URL.Path, status, publicapi.ElapsedMs(start), r.RemoteAddr)
}
