package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/adminaudit"
	integrationsrepo "github.com/lextures/lextures/server/internal/repos/integrations"
	"github.com/lextures/lextures/server/internal/repos/organization"
	integrations "github.com/lextures/lextures/server/internal/service/integrations"
)

// registerIntegrationRoutes wires the plan 16.4 inbound-integration endpoints.
func (d Deps) registerIntegrationRoutes(r chi.Router) {
	r.Get("/api/v1/integrations", d.handleListIntegrations())
	r.Delete("/api/v1/integrations/{id}", d.handleDisconnectIntegration())
	r.Post("/api/v1/integrations/google/import", d.handleGoogleImport())
	r.Get("/api/v1/integrations/{id}/sync-status", d.handleIntegrationSyncStatus())
	r.Get("/integrations/oauth/{provider}/connect", d.handleIntegrationConnect())
	r.Get("/integrations/oauth/{provider}/callback", d.handleIntegrationCallback())
}

// integrationsEnabled returns the wired service, or writes 501 and reports false.
func (d Deps) integrationsEnabled(w http.ResponseWriter) (*integrations.Service, bool) {
	if d.Integrations == nil {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Integrations are not enabled on this environment.")
		return nil, false
	}
	return d.Integrations, true
}

// orgForAdmin resolves the admin's org id after the RBAC check passes.
func (d Deps) orgForAdmin(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (uuid.UUID, bool) {
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
	if err != nil || orgID == uuid.Nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
		return uuid.Nil, false
	}
	return orgID, true
}

// handleListIntegrations is GET /api/v1/integrations.
func (d Deps) handleListIntegrations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		svc, ok := d.integrationsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.orgForAdmin(w, r, userID)
		if !ok {
			return
		}
		views, err := svc.List(r.Context(), orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load integrations.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"integrations": views})
	}
}

// handleDisconnectIntegration is DELETE /api/v1/integrations/{id}.
func (d Deps) handleDisconnectIntegration() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		svc, ok := d.integrationsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.orgForAdmin(w, r, userID)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid integration id.")
			return
		}
		if err := svc.Disconnect(r.Context(), orgID, id); err != nil {
			if errors.Is(err, integrationsrepo.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Integration not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to disconnect integration.")
			return
		}
		d.auditIntegration(r, orgID, userID, "integration_disconnect", id, map[string]any{"connectionId": id})
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleIntegrationConnect is GET /integrations/oauth/{provider}/connect.
func (d Deps) handleIntegrationConnect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		svc, ok := d.integrationsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.orgForAdmin(w, r, userID)
		if !ok {
			return
		}
		provider, err := integrations.ParseProvider(chi.URLParam(r, "provider"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown integration provider.")
			return
		}
		authURL, err := svc.AuthorizeURL(provider, orgID, userID)
		if err != nil {
			if errors.Is(err, integrations.ErrNotConfigured) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This provider has not been configured with OAuth credentials.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start OAuth flow.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"authorizeUrl": authURL})
	}
}

// handleIntegrationCallback is GET /integrations/oauth/{provider}/callback. The
// signed state carries the org/user, so this route is not behind admin auth.
func (d Deps) handleIntegrationCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.integrationsEnabled(w)
		if !ok {
			return
		}
		provider := chi.URLParam(r, "provider")
		q := r.URL.Query()
		if errCode := q.Get("error"); errCode != "" {
			d.redirectToIntegrations(w, r, "error="+errCode)
			return
		}
		conn, err := svc.HandleCallback(r.Context(), provider, q.Get("code"), q.Get("state"))
		if err != nil {
			d.redirectToIntegrations(w, r, "error=connect_failed")
			return
		}
		if conn.ConnectedBy != nil {
			d.auditIntegration(r, conn.OrgID, *conn.ConnectedBy, "integration_connect", conn.ID,
				map[string]any{"provider": conn.Provider, "connectionId": conn.ID})
		}
		d.redirectToIntegrations(w, r, "connected="+conn.Provider)
	}
}

func (d Deps) redirectToIntegrations(w http.ResponseWriter, r *http.Request, query string) {
	// Land the user back on the admin Integrations page with a status query param.
	http.Redirect(w, r, "/admin/integrations?"+query, http.StatusFound)
}

// googleImportRequest is the POST body for /api/v1/integrations/google/import.
type googleImportRequest struct {
	ConnectionID      string `json:"connectionId"`
	LexturesCourseID  string `json:"lexturesCourseId"`
	ExternalCourseID  string `json:"externalCourseId"`
	SyncRoster        bool   `json:"syncRoster"`
	SyncIntervalHours int16  `json:"syncIntervalHours"`
}

// handleGoogleImport is POST /api/v1/integrations/google/import.
func (d Deps) handleGoogleImport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		svc, ok := d.integrationsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.orgForAdmin(w, r, userID)
		if !ok {
			return
		}
		var body googleImportRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		connID, err := uuid.Parse(body.ConnectionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connectionId.")
			return
		}
		courseID, err := uuid.Parse(body.LexturesCourseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid lexturesCourseId.")
			return
		}
		if body.ExternalCourseID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "externalCourseId is required.")
			return
		}
		req := integrations.ImportRequest{
			OrgID:             orgID,
			ConnectionID:      connID,
			LexturesCourseID:  courseID,
			ExternalCourseID:  body.ExternalCourseID,
			SyncRoster:        body.SyncRoster,
			SyncIntervalHours: body.SyncIntervalHours,
		}
		// Enroller is nil for this phase: import records the roster diff and counts
		// without mutating enrollments (the enrollment materialization seam is
		// wired separately). See plan 16.4 §15 rollout phases.
		result, err := svc.Import(r.Context(), orgID, nil, req)
		if err != nil {
			if errors.Is(err, integrationsrepo.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Connection not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Import failed: "+err.Error())
			return
		}
		d.auditIntegration(r, orgID, userID, "integration_import", connID, map[string]any{
			"source":          "google_classroom",
			"recordsImported": result.RecordsImported,
			"recordsSkipped":  result.RecordsSkipped,
			"assignmentCount": result.AssignmentCount,
			"externalCourse":  body.ExternalCourseID,
		})
		writeJSON(w, http.StatusOK, result)
	}
}

// handleIntegrationSyncStatus is GET /api/v1/integrations/{id}/sync-status.
func (d Deps) handleIntegrationSyncStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		svc, ok := d.integrationsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.orgForAdmin(w, r, userID)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid integration id.")
			return
		}
		status, err := svc.SyncStatusFor(r.Context(), orgID, id)
		if err != nil {
			if errors.Is(err, integrationsrepo.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Integration not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load sync status.")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

// auditIntegration writes a best-effort admin audit row for an integration action.
func (d Deps) auditIntegration(r *http.Request, orgID, actorID uuid.UUID, eventType string, targetID uuid.UUID, payload map[string]any) {
	if d.Pool == nil {
		return
	}
	after, _ := json.Marshal(payload)
	targetType := "integration"
	org := orgID
	_, _, _ = adminaudit.Insert(r.Context(), d.Pool, adminaudit.InsertParams{
		OrgID:      &org,
		EventType:  eventType,
		ActorID:    actorID,
		TargetType: &targetType,
		TargetID:   &targetID,
		AfterValue: after,
	})
}
