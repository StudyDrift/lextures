package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/h5p"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/h5pcompletions"
)

type xapiStatementBody struct {
	Statement  json.RawMessage `json:"statement"`
	PackageID  string          `json:"packageId"`
	CourseCode string          `json:"courseCode"`
}

const xapiRateLimitPerMinute = 100

// handlePostXAPIStatements is POST /api/v1/xapi/statements (H5P iframe + future 9.6).
func (d Deps) handlePostXAPIStatements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.h5pEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		var body xapiStatementBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		packageID, err := uuid.Parse(strings.TrimSpace(body.PackageID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid packageId.")
			return
		}
		courseCode := strings.TrimSpace(body.CourseCode)
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseCode is required.")
			return
		}
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		pkg, cid, ok := d.loadH5PForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardH5PAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		n, err := h5pcompletions.CountRecentStatements(r.Context(), d.Pool, packageID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check rate limit.")
			return
		}
		if n >= xapiRateLimitPerMinute {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many xAPI statements; try again later.")
			return
		}
		stmt, err := h5p.ParseStatement(body.Statement)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid xAPI statement.")
			return
		}
		status, scoreRaw, scoreMax := h5p.CompletionStatus(stmt)
		if status == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := h5pcompletions.UpsertFromStatement(r.Context(), d.Pool, packageID, viewer, status, scoreRaw, scoreMax, body.Statement); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record completion.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
