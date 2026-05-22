package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/lrsconfig"
	"github.com/lextures/lextures/server/internal/repos/lrsforwardjobs"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/repos/xapistatements"
	"github.com/lextures/lextures/server/internal/service/learningevents"
)

func (d Deps) emitCourseActivityAsync(userID uuid.UUID, courseCode string, courseID uuid.UUID, kind string, structureItemID *uuid.UUID) {
	if !d.effectiveConfig().XAPIEmissionEnabled || d.Pool == nil {
		return
	}
	if kind != "course_visit" && kind != "content_open" {
		return
	}
	cfg := d.effectiveConfig()
	go func() {
		ctx := context.Background()
		u, err := user.FindByID(ctx, d.Pool, userID)
		if err != nil || u == nil {
			return
		}
		orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
		if err != nil {
			return
		}
		dn := u.Email
		if u.DisplayName != nil {
			dn = *u.DisplayName
		}
		em := learningevents.Emitter{Pool: d.Pool, Cfg: cfg}
		switch kind {
		case "course_visit":
			em.CourseVisited(ctx, orgID, courseID, courseCode, u.Email, dn)
		case "content_open":
			itemID := ""
			if structureItemID != nil {
				itemID = structureItemID.String()
			}
			em.ContentViewed(ctx, orgID, courseID, courseCode, u.Email, dn, itemID, "Course content")
		}
	}()
}

func (d Deps) learningEmitter() learningevents.Emitter {
	return learningevents.Emitter{Pool: d.Pool, Cfg: d.effectiveConfig()}
}

func (d Deps) xapiEmissionEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().XAPIEmissionEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "xAPI emission is not enabled.")
		return false
	}
	return true
}

func (d Deps) adminOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	uid, ok := d.adminRbacUser(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, uid)
	if err != nil || orgID == uuid.Nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
		return uuid.UUID{}, false
	}
	return orgID, true
}

// handleGetAdminLRSConfig is GET /api/v1/admin/lrs-config.
func (d Deps) handleGetAdminLRSConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		orgID, ok := d.adminOrgID(w, r)
		if !ok {
			return
		}
		list, err := lrsconfig.ListByOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load LRS configuration.")
			return
		}
		type row struct {
			ID             string  `json:"id"`
			Label          string  `json:"label"`
			EndpointURL    string  `json:"endpointUrl"`
			AuthType       string  `json:"authType"`
			Username       *string `json:"username,omitempty"`
			Enabled        bool    `json:"enabled"`
			HasPassword    bool    `json:"hasPassword"`
			HasOAuthSecret bool    `json:"hasOauthSecret"`
			OAuthClientID  *string `json:"oauthClientId,omitempty"`
			OAuthTokenURL  *string `json:"oauthTokenUrl,omitempty"`
			UpdatedAt      string  `json:"updatedAt"`
		}
		out := make([]row, 0, len(list))
		for _, e := range list {
			out = append(out, row{
				ID:             e.ID.String(),
				Label:          e.Label,
				EndpointURL:    e.EndpointURL,
				AuthType:       e.AuthType,
				Username:       e.Username,
				Enabled:        e.Enabled,
				HasPassword:    e.HasPassword,
				HasOAuthSecret: e.HasOAuthSecret,
				OAuthClientID:  e.OAuthClientID,
				OAuthTokenURL:  e.OAuthTokenURL,
				UpdatedAt:      e.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePostAdminLRSConfig is POST /api/v1/admin/lrs-config.
func (d Deps) handlePostAdminLRSConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		orgID, ok := d.adminOrgID(w, r)
		if !ok {
			return
		}
		var body struct {
			Label             string `json:"label"`
			EndpointURL       string `json:"endpointUrl"`
			AuthType          string `json:"authType"`
			Username          string `json:"username"`
			Password          string `json:"password"`
			OAuthClientID     string `json:"oauthClientId"`
			OAuthClientSecret string `json:"oauthClientSecret"`
			OAuthTokenURL     string `json:"oauthTokenUrl"`
			Enabled           bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		url := strings.TrimSpace(body.EndpointURL)
		if url == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "endpointUrl is required.")
			return
		}
		authType := strings.ToLower(strings.TrimSpace(body.AuthType))
		if authType != "basic" && authType != "oauth2" {
			authType = "basic"
		}
		cfg := d.effectiveConfig()
		id, err := lrsconfig.Create(r.Context(), d.Pool, cfg.PlatformSecretsKey, lrsconfig.CreateInput{
			OrgID:             orgID,
			Label:             strings.TrimSpace(body.Label),
			EndpointURL:       url,
			AuthType:          authType,
			Username:          strings.TrimSpace(body.Username),
			Password:          []byte(body.Password),
			OAuthClientID:     strings.TrimSpace(body.OAuthClientID),
			OAuthClientSecret: []byte(body.OAuthClientSecret),
			OAuthTokenURL:     strings.TrimSpace(body.OAuthTokenURL),
			Enabled:           body.Enabled,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save LRS endpoint.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// handlePutAdminLRSConfig is PUT /api/v1/admin/lrs-config/{id}.
func (d Deps) handlePutAdminLRSConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		if _, ok := d.adminOrgID(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid endpoint id.")
			return
		}
		var body struct {
			Label             *string `json:"label"`
			EndpointURL       *string `json:"endpointUrl"`
			AuthType          *string `json:"authType"`
			Username          *string `json:"username"`
			Password          *string `json:"password"`
			OAuthClientID     *string `json:"oauthClientId"`
			OAuthClientSecret *string `json:"oauthClientSecret"`
			OAuthTokenURL     *string `json:"oauthTokenUrl"`
			Enabled           *bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := lrsconfig.UpdateInput{
			Label:         body.Label,
			EndpointURL:   body.EndpointURL,
			AuthType:      body.AuthType,
			Username:      body.Username,
			OAuthClientID: body.OAuthClientID,
			OAuthTokenURL: body.OAuthTokenURL,
			Enabled:       body.Enabled,
		}
		if body.Password != nil {
			in.Password = []byte(*body.Password)
		}
		if body.OAuthClientSecret != nil {
			in.OAuthClientSecret = []byte(*body.OAuthClientSecret)
		}
		ok, err := lrsconfig.Update(r.Context(), d.Pool, d.effectiveConfig().PlatformSecretsKey, id, in)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update LRS endpoint.")
			return
		}
		if !ok {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePostAdminLRSTest sends a test statement to an LRS endpoint.
func (d Deps) handlePostAdminLRSTest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		orgID, ok := d.adminOrgID(w, r)
		if !ok {
			return
		}
		if _, err := uuid.Parse(chi.URLParam(r, "id")); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid endpoint id.")
			return
		}
		emitter := d.learningEmitter()
		emitter.LoggedIn(r.Context(), orgID, "lrs-test@lextures.local", "LRS Test")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Test statement stored locally; forwarding job queued if endpoint is enabled.",
		})
	}
}

// handleGetAdminLRSDeadLetter is GET /api/v1/admin/lrs-dead-letter.
func (d Deps) handleGetAdminLRSDeadLetter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		if _, ok := d.adminOrgID(w, r); !ok {
			return
		}
		rows, err := lrsforwardjobs.ListDeadLetter(r.Context(), d.Pool, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load dead-letter queue.")
			return
		}
		type row struct {
			ID          string  `json:"id"`
			StatementID string  `json:"statementId"`
			EndpointID  string  `json:"lrsEndpointId"`
			LastError   *string `json:"lastError,omitempty"`
			CreatedAt   string  `json:"createdAt"`
		}
		out := make([]row, 0, len(rows))
		for _, dl := range rows {
			out = append(out, row{
				ID:          dl.ID.String(),
				StatementID: dl.StatementID.String(),
				EndpointID:  dl.LRSEndpointID.String(),
				LastError:   dl.LastError,
				CreatedAt:   dl.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePostAdminLRSDeadLetterRetry is POST /api/v1/admin/lrs-dead-letter/{id}/retry.
func (d Deps) handlePostAdminLRSDeadLetterRetry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		if _, ok := d.adminOrgID(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		ok, err := lrsforwardjobs.RetryDeadLetter(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to retry.")
			return
		}
		if !ok {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetCourseEvents is GET /api/v1/courses/{course_code}/events.
func (d Deps) handleGetCourseEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.xapiEmissionEnabled(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view the event log.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		now := time.Now().UTC()
		since := now.Add(-7 * 24 * time.Hour)
		if q := strings.TrimSpace(r.URL.Query().Get("since")); q != "" {
			if t, err := time.Parse(time.RFC3339, q); err == nil {
				since = t.UTC()
			}
		}
		rows, err := xapistatements.ListForCourse(r.Context(), d.Pool, *cid, since, now, 500)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load events.")
			return
		}
		type item struct {
			StatementID string          `json:"statementId"`
			Verb        string          `json:"verb"`
			ObjectID    string          `json:"objectId"`
			ObjectTitle *string         `json:"objectTitle,omitempty"`
			StoredAt    string          `json:"storedAt"`
			FullJSON    json.RawMessage `json:"fullJson"`
		}
		out := make([]item, 0, len(rows))
		for _, row := range rows {
			title := row.ObjectTitle
			out = append(out, item{
				StatementID: row.StatementID.String(),
				Verb:        row.VerbID,
				ObjectID:    row.ObjectID,
				ObjectTitle: title,
				StoredAt:    row.StoredAt.UTC().Format(time.RFC3339),
				FullJSON:    row.FullJSON,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": out})
	}
}
