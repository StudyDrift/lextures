package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerAdminContentToolsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/content-tools/versions", d.handleAdminContentToolsVersionsList())
	r.Patch("/api/v1/admin/content-tools/versions/{tool_id}/{version}", d.handleAdminContentToolsVersionPatch())
	r.Post("/api/v1/admin/content-tools/migrations", d.handleAdminContentToolsMigrationCreate())
	r.Get("/api/v1/admin/content-tools/migrations/{job_id}", d.handleAdminContentToolsMigrationGet())
	r.Get("/api/v1/admin/content-tools/quarantine", d.handleAdminContentToolsQuarantineList())
}

func (d Deps) handleAdminContentToolsVersionsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		// Ensure mirror is populated even if boot sync was skipped (tests).
		_ = ctsvc.SyncRegistryMirror(r.Context(), d.Pool, ctsvc.MustDefault())
		toolID := strings.TrimSpace(r.URL.Query().Get("toolId"))
		rows, err := ctrepo.ListToolVersions(r.Context(), d.Pool, toolID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list tool versions.")
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			breakerOpen := row.BreakerOpenAt != nil || ctsvc.DefaultBreaker().IsOpen(row.ToolID)
			item := map[string]any{
				"toolId":              row.ToolID,
				"version":             row.Version,
				"configSchemaVersion": row.ConfigSchemaVersion,
				"stateSchemaVersion":  row.StateSchemaVersion,
				"sandboxMode":         row.SandboxMode,
				"status":              row.Status,
				"breakerOpen":         breakerOpen,
				"breakerOpenAt":       row.BreakerOpenAt,
				"sunsetAt":            row.SunsetAt,
				"firstSeenAt":         row.FirstSeenAt,
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"versions":           items,
			"sandboxMode":        ctsvc.PlatformSandboxMode(),
			"contractSupported":  []int{ctsvc.SupportedContractMin, ctsvc.SupportedContractMax},
		})
	}
}

type adminContentToolsVersionPatchBody struct {
	Status       *string `json:"status"`
	ResetBreaker bool    `json:"resetBreaker"`
	OpenBreaker  bool    `json:"openBreaker"`
}

func (d Deps) handleAdminContentToolsVersionPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			jobsMethodNotAllowed(w, http.MethodPatch)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		toolID := chi.URLParam(r, "tool_id")
		version := chi.URLParam(r, "version")
		var body adminContentToolsVersionPatchBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Status != nil {
			switch strings.TrimSpace(*body.Status) {
			case "active", "deprecated", "sunset", "disabled":
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be active, deprecated, sunset, or disabled.")
				return
			}
		}
		// Ensure row exists.
		_ = ctsvc.SyncRegistryMirror(r.Context(), d.Pool, ctsvc.MustDefault())
		row, err := ctrepo.PatchToolVersion(r.Context(), d.Pool, toolID, version, ctrepo.VersionPatch{
			Status:       body.Status,
			ResetBreaker: body.ResetBreaker,
			OpenBreaker:  body.OpenBreaker,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update tool version.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tool version not found.")
			return
		}
		if body.ResetBreaker {
			ctsvc.DefaultBreaker().Reset(toolID)
		}
		if body.OpenBreaker {
			ctsvc.DefaultBreaker().Open(toolID, time.Now().UTC())
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"toolId":        row.ToolID,
			"version":       row.Version,
			"status":        row.Status,
			"breakerOpen":   row.BreakerOpenAt != nil || ctsvc.DefaultBreaker().IsOpen(toolID),
			"breakerOpenAt": row.BreakerOpenAt,
			"sunsetAt":      row.SunsetAt,
		})
	}
}

type adminContentToolsMigrationCreateBody struct {
	ToolID      string `json:"toolId"`
	FromVersion int    `json:"fromVersion"`
	ToVersion   int    `json:"toVersion"`
	DryRun      *bool  `json:"dryRun"`
}

func (d Deps) handleAdminContentToolsMigrationCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jobsMethodNotAllowed(w, http.MethodPost)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		var body adminContentToolsMigrationCreateBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		toolID := strings.TrimSpace(body.ToolID)
		if toolID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "toolId is required.")
			return
		}
		if body.FromVersion <= 0 {
			body.FromVersion = 1
		}
		if body.ToVersion <= body.FromVersion {
			body.ToVersion = ctsvc.DefaultMigrations().CurrentStateSchemaVersion(toolID)
		}
		if body.ToVersion <= body.FromVersion {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "toVersion must be greater than fromVersion.")
			return
		}
		dryRun := true
		if body.DryRun != nil {
			dryRun = *body.DryRun
		}
		job, err := ctrepo.CreateMigrationJob(r.Context(), d.Pool, toolID, body.FromVersion, body.ToVersion, dryRun)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create migration job.")
			return
		}
		// Run inline; jobs are batched and resumable via cursor if interrupted mid-flight.
		if err := ctsvc.RunMigrationJob(r.Context(), d.Pool, job.ID); err != nil {
			fresh, _ := ctrepo.GetMigrationJob(r.Context(), d.Pool, job.ID)
			if fresh == nil {
				fresh = job
			}
			writeJSON(w, http.StatusAccepted, migrationJobJSON(fresh))
			return
		}
		fresh, _ := ctrepo.GetMigrationJob(r.Context(), d.Pool, job.ID)
		if fresh == nil {
			fresh = job
		}
		writeJSON(w, http.StatusAccepted, migrationJobJSON(fresh))
	}
}

func (d Deps) handleAdminContentToolsMigrationGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		job, err := ctrepo.GetMigrationJob(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load migration job.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Migration job not found.")
			return
		}
		writeJSON(w, http.StatusOK, migrationJobJSON(job))
	}
}

func (d Deps) handleAdminContentToolsQuarantineList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		toolID := strings.TrimSpace(r.URL.Query().Get("toolId"))
		rows, err := ctrepo.ListQuarantine(r.Context(), d.Pool, toolID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list quarantine.")
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, map[string]any{
				"id":          row.ID,
				"stateId":     row.StateID,
				"toolId":      row.ToolID,
				"fromVersion": row.FromVersion,
				"toVersion":   row.ToVersion,
				"error":       row.Error,
				"createdAt":   row.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func migrationJobJSON(job *ctrepo.MigrationJobRow) map[string]any {
	if job == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":           job.ID,
		"toolId":       job.ToolID,
		"fromVersion":  job.FromVersion,
		"toVersion":    job.ToVersion,
		"dryRun":       job.DryRun,
		"status":       job.Status,
		"totalDocs":    job.TotalDocs,
		"migratedDocs": job.MigratedDocs,
		"failedDocs":   job.FailedDocs,
		"error":        job.Error,
		"createdAt":    job.CreatedAt,
		"finishedAt":   job.FinishedAt,
	}
}
