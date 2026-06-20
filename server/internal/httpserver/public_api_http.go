package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/publicapi"
	"github.com/lextures/lextures/server/internal/repos/apirequestlog"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	publicapirepo "github.com/lextures/lextures/server/internal/repos/publicapi"
)

const publicAPIQuotaPerMinute = 1000

type publicAPIContext struct {
	start   time.Time
	tokenID *uuid.UUID
	userID  uuid.UUID
	logged  bool
}

func (d Deps) registerPublicAPIRoutes(r chi.Router) {
	r.Get("/api/v1/openapi.json", d.handlePublicAPIOpenAPI())
	r.Get("/api/v1/docs", d.handlePublicAPIDocs())
	r.Get("/api/v1/redoc", d.handlePublicAPIRedoc())
	r.Get("/api/v1/graphql", d.handlePublicAPIGraphQL())
	r.Get("/api/v1/assignments", d.handlePublicAPIListAssignments())
	r.Get("/api/v1/grades", d.handlePublicAPIListGrades())
}

func (d Deps) publicAPIEnabled() bool {
	return d.effectiveConfig().FFPublicAPI
}

func (d Deps) apiDocsEnabled() bool {
	cfg := d.effectiveConfig()
	return cfg.EnableAPIDocs || cfg.FFAPIDocs
}

func (d Deps) publicAPIServiceUnavailable(w http.ResponseWriter, r *http.Request) bool {
	if d.publicAPIEnabled() {
		return false
	}
	w.Header().Set("Retry-After", "300")
	publicapi.WriteProblem(w, publicapi.Problem{
		Type:     "service-unavailable",
		Title:    "Service Unavailable",
		Status:   http.StatusServiceUnavailable,
		Detail:   "The public API is not enabled on this deployment.",
		Instance: r.URL.Path,
	})
	return true
}

func (d Deps) beginPublicAPI(w http.ResponseWriter, r *http.Request) (*publicAPIContext, *auth.APITokenAuth, bool) {
	if d.publicAPIServiceUnavailable(w, r) {
		return nil, nil, false
	}
	if _, ok := auth.BearerToken(r.Header); !ok {
		publicapi.Unauthorized(w, r.URL.Path, "")
		return nil, nil, false
	}
	if d.JWTSigner == nil {
		publicapi.WriteProblem(w, publicapi.Problem{
			Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError,
			Detail: "Authentication is not configured.", Instance: r.URL.Path,
		})
		return nil, nil, false
	}
	u, ctx, err := auth.UserFromRequestOrAccessKey(r, d.JWTSigner, d.Pool)
	if err != nil {
		publicapi.Unauthorized(w, r.URL.Path, "")
		return nil, nil, false
	}
	*r = *r.WithContext(ctx)
	userID, err := uuid.Parse(u.UserID)
	if err != nil {
		publicapi.Unauthorized(w, r.URL.Path, "")
		return nil, nil, false
	}
	if _, ok := d.validateMeUser(w, r, u, userID); !ok {
		return nil, nil, false
	}
	tok, isToken := auth.APITokenFromContext(ctx)
	if !isToken {
		publicapi.Unauthorized(w, r.URL.Path, "Institutional API tokens are required for this endpoint.")
		return nil, nil, false
	}
	if allowed, retry := publicapi.AllowToken(tok.TokenID.String(), publicAPIQuotaPerMinute); !allowed {
		publicapi.RateLimited(w, r.URL.Path, retry)
		return nil, nil, false
	}
	pctx := &publicAPIContext{start: time.Now(), userID: userID}
	id := tok.TokenID
	pctx.tokenID = &id
	return pctx, tok, true
}

func (d Deps) finishPublicAPILog(pctx *publicAPIContext, r *http.Request, status int) {
	if pctx == nil || pctx.logged || d.Pool == nil {
		return
	}
	pctx.logged = true
	latency := int(time.Since(pctx.start).Milliseconds())
	uid := pctx.userID
	apirequestlog.LogAsync(d.Pool, pctx.tokenID, &uid, r.Method, r.URL.Path, status, latency, r.RemoteAddr, d.effectiveConfig().JWTSecret)
}

func (d Deps) requirePublicAPIScope(w http.ResponseWriter, r *http.Request, pctx *publicAPIContext, tok *auth.APITokenAuth, scope string) bool {
	if tok == nil || !publicapi.HasScope(tok.Scopes, scope) {
		publicapi.ForbiddenScope(w, r.URL.Path, scope)
		d.finishPublicAPILog(pctx, r, http.StatusForbidden)
		return false
	}
	return true
}

func (d Deps) publicAPIListCourses(w http.ResponseWriter, r *http.Request, pctx *publicAPIContext, tok *auth.APITokenAuth) {
	if !d.requirePublicAPIScope(w, r, pctx, tok, "courses:read") {
		return
	}
	offset, err := publicapi.DecodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "invalid-input", Title: "Invalid cursor", Status: http.StatusBadRequest, Detail: err.Error(), Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusBadRequest)
		return
	}
	limit, err := publicapi.ParseLimit(r.URL.Query().Get("limit"), 25, 100)
	if err != nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "invalid-input", Title: "Invalid limit", Status: http.StatusBadRequest, Detail: err.Error(), Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusBadRequest)
		return
	}
	courses, err := course.ListForEnrolledUser(r.Context(), d.Pool, pctx.userID, nil)
	if err != nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Detail: "Failed to list courses.", Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusInternalServerError)
		return
	}
	if len(tok.CourseIDs) > 0 {
		allowed := make(map[string]struct{}, len(tok.CourseIDs))
		for _, id := range tok.CourseIDs {
			allowed[id.String()] = struct{}{}
		}
		filtered := make([]course.CoursePublic, 0, len(courses))
		for _, c := range courses {
			if _, ok := allowed[c.ID]; ok {
				filtered = append(filtered, c)
			}
		}
		courses = filtered
	}
	fields := publicapi.ParseSparseFields(r.URL.Query(), "courses")
	page, nextCursor := publicapi.PaginateSlice(courses, offset, limit)
	data := make([]map[string]any, 0, len(page))
	for _, c := range page {
		obj := map[string]any{
			"id": c.ID, "courseCode": c.CourseCode, "title": c.Title, "description": c.Description,
			"published": c.Published, "archived": c.Archived,
		}
		data = append(data, publicapi.FilterObject(fields, obj))
	}
	publicapi.WriteCollection(w, http.StatusOK, publicapi.CollectionResponse{
		Data:  data,
		Meta:  publicapi.CollectionMeta{Total: len(courses), Cursor: nextCursor},
		Links: publicapi.BuildPageLinks("/api/v1/courses", r.URL.Query(), offset, limit, len(courses)),
	})
	d.finishPublicAPILog(pctx, r, http.StatusOK)
}

func (d Deps) publicAPIGetCourseByID(w http.ResponseWriter, r *http.Request, pctx *publicAPIContext, tok *auth.APITokenAuth, courseID uuid.UUID) {
	if !d.requirePublicAPIScope(w, r, pctx, tok, "courses:read") {
		return
	}
	if !auth.AccessKeyAllowsCourse(r.Context(), courseID) {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Detail: "Course not found.", Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	code, err := course.GetCourseCodeByID(r.Context(), d.Pool, courseID)
	if err != nil || code == nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusInternalServerError)
		return
	}
	has, err := enrollment.UserHasAccess(r.Context(), d.Pool, *code, pctx.userID)
	if err != nil || !has {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Detail: "Course not found.", Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	crow, err := course.GetPublicByID(r.Context(), d.Pool, courseID)
	if err != nil || crow == nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Detail: "Course not found.", Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": crow})
	d.finishPublicAPILog(pctx, r, http.StatusOK)
}

func (d Deps) publicAPIListEnrollments(w http.ResponseWriter, r *http.Request, pctx *publicAPIContext, tok *auth.APITokenAuth, courseID uuid.UUID) {
	if !d.requirePublicAPIScope(w, r, pctx, tok, "enrollments:read") {
		return
	}
	code, err := course.GetCourseCodeByID(r.Context(), d.Pool, courseID)
	if err != nil || code == nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	if !auth.AccessKeyAllowsCourse(r.Context(), courseID) {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	has, err := enrollment.UserHasAccess(r.Context(), d.Pool, *code, pctx.userID)
	if err != nil || !has {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	roster, err := enrollment.ListRosterForCourse(r.Context(), d.Pool, *code)
	if err != nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusInternalServerError)
		return
	}
	offset, _ := publicapi.DecodeCursor(r.URL.Query().Get("cursor"))
	limit, _ := publicapi.ParseLimit(r.URL.Query().Get("limit"), 25, 100)
	page, nextCursor := publicapi.PaginateSlice(roster, offset, limit)
	data := make([]map[string]any, 0, len(page))
	for _, e := range page {
		data = append(data, map[string]any{
			"id": e.ID.String(), "userId": e.UserID.String(), "role": e.Role,
			"displayName": e.DisplayName,
		})
	}
	publicapi.WriteCollection(w, http.StatusOK, publicapi.CollectionResponse{
		Data:  data,
		Meta:  publicapi.CollectionMeta{Total: len(roster), Cursor: nextCursor},
		Links: publicapi.BuildPageLinks(r.URL.Path, r.URL.Query(), offset, limit, len(roster)),
	})
	d.finishPublicAPILog(pctx, r, http.StatusOK)
}

func (d Deps) publicAPIGetUser(w http.ResponseWriter, r *http.Request, pctx *publicAPIContext, tok *auth.APITokenAuth, targetID uuid.UUID) {
	if !d.requirePublicAPIScope(w, r, pctx, tok, "users:read") {
		return
	}
	ok, err := publicapirepo.ViewerMayReadUser(r.Context(), d.Pool, pctx.userID, targetID)
	if err != nil || !ok {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	u, err := publicapirepo.GetUserByID(r.Context(), d.Pool, targetID)
	if err != nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
		d.finishPublicAPILog(pctx, r, http.StatusNotFound)
		return
	}
	if !publicapi.HasScope(tok.Scopes, "pii:read") {
		u.Email = ""
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": u})
	d.finishPublicAPILog(pctx, r, http.StatusOK)
}

// serveIfPublicAPIToken authenticates API-token requests when the public API flag is on.
func (d Deps) serveIfPublicAPIToken(w http.ResponseWriter, r *http.Request, serve func(*publicAPIContext, *auth.APITokenAuth)) bool {
	if !d.publicAPIEnabled() {
		return false
	}
	if _, ok := auth.BearerToken(r.Header); !ok {
		publicapi.Unauthorized(w, r.URL.Path, "")
		return true
	}
	if d.JWTSigner == nil {
		publicapi.WriteProblem(w, publicapi.Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Instance: r.URL.Path})
		return true
	}
	u, ctx, err := auth.UserFromRequestOrAccessKey(r, d.JWTSigner, d.Pool)
	if err != nil {
		publicapi.Unauthorized(w, r.URL.Path, "")
		return true
	}
	*r = *r.WithContext(ctx)
	userID, err := uuid.Parse(u.UserID)
	if err != nil {
		publicapi.Unauthorized(w, r.URL.Path, "")
		return true
	}
	if _, ok := d.validateMeUser(w, r, u, userID); !ok {
		return true
	}
	tok, isToken := auth.APITokenFromContext(ctx)
	if !isToken {
		return false
	}
	if allowed, retry := publicapi.AllowToken(tok.TokenID.String(), publicAPIQuotaPerMinute); !allowed {
		publicapi.RateLimited(w, r.URL.Path, retry)
		return true
	}
	id := tok.TokenID
	pctx := &publicAPIContext{start: time.Now(), userID: userID, tokenID: &id}
	serve(pctx, tok)
	return true
}

func (d Deps) handlePublicAPIListAssignments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pctx, tok, ok := d.beginPublicAPI(w, r)
		if !ok {
			return
		}
		if !d.requirePublicAPIScope(w, r, pctx, tok, "assignments:read") {
			return
		}
		items, err := publicapirepo.ListAssignmentsForUser(r.Context(), d.Pool, pctx.userID, tok.CourseIDs)
		if err != nil {
			publicapi.WriteProblem(w, publicapi.Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Instance: r.URL.Path})
			d.finishPublicAPILog(pctx, r, http.StatusInternalServerError)
			return
		}
		offset, _ := publicapi.DecodeCursor(r.URL.Query().Get("cursor"))
		limit, _ := publicapi.ParseLimit(r.URL.Query().Get("limit"), 25, 100)
		page, nextCursor := publicapi.PaginateSlice(items, offset, limit)
		publicapi.WriteCollection(w, http.StatusOK, publicapi.CollectionResponse{
			Data: page, Meta: publicapi.CollectionMeta{Total: len(items), Cursor: nextCursor},
			Links: publicapi.BuildPageLinks("/api/v1/assignments", r.URL.Query(), offset, limit, len(items)),
		})
		d.finishPublicAPILog(pctx, r, http.StatusOK)
	}
}

func (d Deps) handlePublicAPIListGrades() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pctx, tok, ok := d.beginPublicAPI(w, r)
		if !ok {
			return
		}
		if !d.requirePublicAPIScope(w, r, pctx, tok, "grades:read") {
			return
		}
		items, err := publicapirepo.ListGradesForUser(r.Context(), d.Pool, pctx.userID, tok.CourseIDs)
		if err != nil {
			publicapi.WriteProblem(w, publicapi.Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Instance: r.URL.Path})
			d.finishPublicAPILog(pctx, r, http.StatusInternalServerError)
			return
		}
		offset, _ := publicapi.DecodeCursor(r.URL.Query().Get("cursor"))
		limit, _ := publicapi.ParseLimit(r.URL.Query().Get("limit"), 25, 100)
		page, nextCursor := publicapi.PaginateSlice(items, offset, limit)
		publicapi.WriteCollection(w, http.StatusOK, publicapi.CollectionResponse{
			Data: page, Meta: publicapi.CollectionMeta{Total: len(items), Cursor: nextCursor},
			Links: publicapi.BuildPageLinks("/api/v1/grades", r.URL.Query(), offset, limit, len(items)),
		})
		d.finishPublicAPILog(pctx, r, http.StatusOK)
	}
}

func (d Deps) handlePublicAPIOpenAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write([]byte(publicapi.OpenAPI31Document))
	}
}

func (d Deps) handlePublicAPIDocs() http.HandlerFunc {
	const html = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"/><title>Lextures API</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/></head>
<body><div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>window.onload=function(){SwaggerUIBundle({url:'/api/v1/openapi.json',dom_id:'#swagger-ui'});};</script>
</body></html>`
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.apiDocsEnabled() {
			publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}
}

func (d Deps) handlePublicAPIRedoc() http.HandlerFunc {
	const html = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"/><title>Lextures API</title>
<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script></head>
<body><redoc spec-url="/api/v1/openapi.json"></redoc></body></html>`
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.apiDocsEnabled() {
			publicapi.WriteProblem(w, publicapi.Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Instance: r.URL.Path})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}
}

func (d Deps) handlePublicAPIGraphQL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pctx, tok, ok := d.beginPublicAPI(w, r)
		if !ok {
			return
		}
		if !d.requirePublicAPIScope(w, r, pctx, tok, "graphql:read") {
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(publicapi.GraphQLSchemaStub))
		d.finishPublicAPILog(pctx, r, http.StatusOK)
	}
}
