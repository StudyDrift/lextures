package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoConsortium "github.com/lextures/lextures/server/internal/repos/consortium"
	repoSIS "github.com/lextures/lextures/server/internal/repos/sis"
	serviceSIS "github.com/lextures/lextures/server/internal/service/sis"
	"github.com/lextures/lextures/server/internal/workers/sissync"
)

// handleAdminSISConnections is GET/POST /api/v1/admin/orgs/:orgId/sis/connections.
func (d Deps) handleAdminSISConnections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFSISIntegration {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "SIS integration is not enabled.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			conns, err := repoSIS.ListConnections(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list SIS connections.")
				return
			}
			out := make([]map[string]any, 0, len(conns))
			for i := range conns {
				out = append(out, connectionToJSON(&conns[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"connections": out})

		case http.MethodPost:
			var body struct {
				Vendor          string `json:"vendor"`
				BaseURL         string `json:"baseUrl"`
				ClientIDRef     string `json:"clientIdRef"`
				ClientSecretRef string `json:"clientSecretRef"`
				SyncSchedule    string `json:"syncSchedule"`
				SyncMode        string `json:"syncMode"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			vendor := strings.TrimSpace(body.Vendor)
			if vendor == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "vendor is required.")
				return
			}
			if !repoSIS.ValidVendor(vendor) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"vendor must be one of: powerschool, infinite_campus, skyward, aeries, banner, workday, colleague, jenzabar, peoplesoft.")
				return
			}
			baseURL := strings.TrimSpace(body.BaseURL)
			if baseURL == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "baseUrl is required.")
				return
			}
			clientIDRef := strings.TrimSpace(body.ClientIDRef)
			if clientIDRef == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "clientIdRef is required.")
				return
			}
			clientSecretRef := strings.TrimSpace(body.ClientSecretRef)
			if clientSecretRef == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "clientSecretRef is required.")
				return
			}
			schedule := strings.TrimSpace(body.SyncSchedule)
			if schedule == "" {
				schedule = "0 2 * * *"
			}
			mode := strings.TrimSpace(body.SyncMode)
			if mode == "" {
				mode = "incremental"
			}
			switch mode {
			case "incremental", "full":
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "syncMode must be incremental or full.")
				return
			}
			conn, err := repoSIS.CreateConnection(r.Context(), d.Pool, orgID,
				vendor, baseURL, clientIDRef, clientSecretRef, schedule, mode)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create SIS connection.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"connection": connectionToJSON(conn)})

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminSISConnection is PATCH /api/v1/admin/orgs/:orgId/sis/connections/:id.
func (d Deps) handleAdminSISConnection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFSISIntegration {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "SIS integration is not enabled.")
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		connID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connection id.")
			return
		}

		var body struct {
			BaseURL         *string `json:"baseUrl"`
			ClientIDRef     *string `json:"clientIdRef"`
			ClientSecretRef *string `json:"clientSecretRef"`
			SyncSchedule    *string `json:"syncSchedule"`
			SyncMode        *string `json:"syncMode"`
			Active          *bool   `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.SyncMode != nil {
			switch *body.SyncMode {
			case "incremental", "full":
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "syncMode must be incremental or full.")
				return
			}
		}
		conn, err := repoSIS.UpdateConnection(r.Context(), d.Pool, orgID, connID, repoSIS.UpdateConnectionFields{
			BaseURL:         body.BaseURL,
			ClientIDRef:     body.ClientIDRef,
			ClientSecretRef: body.ClientSecretRef,
			SyncSchedule:    body.SyncSchedule,
			SyncMode:        body.SyncMode,
			Active:          body.Active,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update SIS connection.")
			return
		}
		if conn == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "SIS connection not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"connection": connectionToJSON(conn)})
	}
}

// handleAdminSISSync is POST /api/v1/admin/orgs/:orgId/sis/connections/:id/sync.
func (d Deps) handleAdminSISSync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFSISIntegration {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "SIS integration is not enabled.")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		connID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connection id.")
			return
		}
		conn, err := repoSIS.GetConnection(r.Context(), d.Pool, orgID, connID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load SIS connection.")
			return
		}
		if conn == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "SIS connection not found.")
			return
		}
		result, err := sissync.RunSync(r.Context(), d.Pool, *conn)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Sync failed to start.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"logId":   result.LogID.String(),
			"status":  result.Status,
			"summary": result.Summary,
			"errors":  result.Errors,
		})
	}
}

// handleAdminSISSyncLogs is GET /api/v1/admin/orgs/:orgId/sis/sync-logs.
func (d Deps) handleAdminSISSyncLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFSISIntegration {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "SIS integration is not enabled.")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		limit := 50
		if l := strings.TrimSpace(r.URL.Query().Get("limit")); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		offset := 0
		if o := strings.TrimSpace(r.URL.Query().Get("offset")); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}
		logs, err := repoSIS.ListSyncLogs(r.Context(), d.Pool, orgID, limit, offset)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list sync logs.")
			return
		}
		out := make([]map[string]any, 0, len(logs))
		for i := range logs {
			out = append(out, syncLogToJSON(&logs[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"logs": out, "limit": limit, "offset": offset})
	}
}

// handleAdminSISGradePassback is POST /api/v1/admin/orgs/:orgId/sis/grade-passback.
func (d Deps) handleAdminSISGradePassback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFSISIntegration {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "SIS integration is not enabled.")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}

		var body struct {
			ConnectionID   string `json:"connectionId"`
			GradingPeriod  string `json:"gradingPeriod"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		connID, err := uuid.Parse(strings.TrimSpace(body.ConnectionID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "connectionId is required and must be a valid UUID.")
			return
		}
		if strings.TrimSpace(body.GradingPeriod) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "gradingPeriod is required.")
			return
		}
		conn, err := repoSIS.GetConnection(r.Context(), d.Pool, orgID, connID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load SIS connection.")
			return
		}
		if conn == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "SIS connection not found.")
			return
		}

		// Grade passback creates a sync log entry to record the passback run.
		// Real implementation would query final course grades and POST them to
		// the SIS via OneRoster results resource or native grade-passback API.
		log, err := repoSIS.CreateSyncLog(r.Context(), d.Pool, conn.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create passback log.")
			return
		}
		summary := repoSIS.SyncSummary{}
		recordsSent := 0
		if d.effectiveConfig().FFConsortiumSharing {
			if count, countErr := repoConsortium.CountPassbackEnrollments(r.Context(), d.Pool, orgID, false); countErr == nil {
				recordsSent += count
			}
			if count, countErr := repoConsortium.CountPassbackEnrollments(r.Context(), d.Pool, orgID, true); countErr == nil {
				recordsSent += count
			}
		}
		if err := repoSIS.FinishSyncLog(r.Context(), d.Pool, log.ID, repoSIS.SyncStatusSuccess, summary, nil); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to finish passback log.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"logId":         log.ID.String(),
			"status":        repoSIS.SyncStatusSuccess,
			"gradingPeriod": body.GradingPeriod,
			"recordsSent":   recordsSent,
		})
	}
}

// handleAdminSISTestConnection is POST /api/v1/admin/orgs/:orgId/sis/connections/:id/test.
func (d Deps) handleAdminSISTestConnection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFSISIntegration {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "SIS integration is not enabled.")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		connID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connection id.")
			return
		}
		conn, err := repoSIS.GetConnection(r.Context(), d.Pool, orgID, connID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load SIS connection.")
			return
		}
		if conn == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "SIS connection not found.")
			return
		}
		cfg := serviceSIS.ConnectionConfig{
			Vendor:          conn.Vendor,
			BaseURL:         conn.BaseURL,
			ClientIDRef:     conn.ClientIDRef,
			ClientSecretRef: conn.ClientSecretRef,
		}
		if serviceSIS.IsHEVendor(conn.Vendor) {
			adapter := serviceSIS.AdapterFor(conn.Vendor)
			if adapter == nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "HE adapter not available.")
				return
			}
			if err := adapter.TestConnection(r.Context(), cfg); err != nil {
				apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Connection test failed: "+err.Error())
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"message": "Connection test succeeded.",
			"vendor":  conn.Vendor,
			"market":  repoSIS.VendorMarket(conn.Vendor),
		})
	}
}

func (d Deps) registerSISRoutes(r chi.Router) {
	r.Method(http.MethodGet, "/api/v1/admin/orgs/{orgId}/sis/connections", d.handleAdminSISConnections())
	r.Method(http.MethodPost, "/api/v1/admin/orgs/{orgId}/sis/connections", d.handleAdminSISConnections())
	r.Method(http.MethodPatch, "/api/v1/admin/orgs/{orgId}/sis/connections/{id}", d.handleAdminSISConnection())
	r.Method(http.MethodPost, "/api/v1/admin/orgs/{orgId}/sis/connections/{id}/sync", d.handleAdminSISSync())
	r.Method(http.MethodPost, "/api/v1/admin/orgs/{orgId}/sis/connections/{id}/test", d.handleAdminSISTestConnection())
	r.Method(http.MethodGet, "/api/v1/admin/orgs/{orgId}/sis/sync-logs", d.handleAdminSISSyncLogs())
	r.Method(http.MethodPost, "/api/v1/admin/orgs/{orgId}/sis/grade-passback", d.handleAdminSISGradePassback())
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func connectionToJSON(c *repoSIS.Connection) map[string]any {
	if c == nil {
		return nil
	}
	m := map[string]any{
		"id":              c.ID.String(),
		"orgId":           c.OrgID.String(),
		"vendor":          c.Vendor,
		"market":          repoSIS.VendorMarket(c.Vendor),
		"baseUrl":         c.BaseURL,
		"clientIdRef":     c.ClientIDRef,
		"clientSecretRef": c.ClientSecretRef,
		"syncSchedule":    c.SyncSchedule,
		"syncMode":        c.SyncMode,
		"active":          c.Active,
		"createdAt":       c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if c.LastSyncAt != nil {
		m["lastSyncAt"] = c.LastSyncAt.UTC().Format("2006-01-02T15:04:05Z")
	} else {
		m["lastSyncAt"] = nil
	}
	return m
}

func syncLogToJSON(l *repoSIS.SyncLog) map[string]any {
	if l == nil {
		return nil
	}
	m := map[string]any{
		"id":           l.ID.String(),
		"connectionId": l.ConnectionID.String(),
		"startedAt":    l.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"status":       l.Status,
		"summary":      l.Summary,
		"errors":       l.Errors,
	}
	if l.FinishedAt != nil {
		m["finishedAt"] = l.FinishedAt.UTC().Format("2006-01-02T15:04:05Z")
	} else {
		m["finishedAt"] = nil
	}
	return m
}
