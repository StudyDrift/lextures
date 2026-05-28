package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	backupservice "github.com/lextures/lextures/server/internal/service/backup"
)

func (d Deps) backupModuleEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().BackupModuleEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Backup module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireBackupAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := backupservice.CheckAdmin(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, false
	}
	if !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return uid, true
}

func (d Deps) registerBackupOpsRoutes(r chi.Router) {
	r.Get("/api/v1/internal/ops/backup-status", d.handleGetBackupStatus())
	r.Post("/api/v1/internal/ops/restore-drill", d.handlePostRestoreDrill())
}

// GET /api/v1/internal/ops/backup-status
func (d Deps) handleGetBackupStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.backupModuleEnabled(w) {
			return
		}
		if _, ok := d.requireBackupAdmin(w, r); !ok {
			return
		}
		status, err := backupservice.GetBackupStatus(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load backup status.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(status)
	}
}

type postRestoreDrillBody struct {
	DrillDate          string  `json:"drillDate"`          // YYYY-MM-DD
	BackupTimestamp    string  `json:"backupTimestamp"`    // RFC3339
	RestoreStart       string  `json:"restoreStart"`       // RFC3339
	RestoreEnd         *string `json:"restoreEnd"`         // RFC3339
	RPOAchievedMinutes *int    `json:"rpoAchievedMinutes"`
	RTOAchievedMinutes *int    `json:"rtoAchievedMinutes"`
	Pass               *bool   `json:"pass"`
	SmokeTestOutput    *string `json:"smokeTestOutput"`
	Notes              *string `json:"notes"`
}

// POST /api/v1/internal/ops/restore-drill
func (d Deps) handlePostRestoreDrill() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.backupModuleEnabled(w) {
			return
		}
		conductorID, ok := d.requireBackupAdmin(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postRestoreDrillBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		drillDate, err := time.Parse("2006-01-02", body.DrillDate)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "drillDate must be YYYY-MM-DD.")
			return
		}
		backupTS, err := time.Parse(time.RFC3339, body.BackupTimestamp)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "backupTimestamp must be RFC3339.")
			return
		}
		restoreStart, err := time.Parse(time.RFC3339, body.RestoreStart)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "restoreStart must be RFC3339.")
			return
		}
		var restoreEnd *time.Time
		if body.RestoreEnd != nil {
			t, err := time.Parse(time.RFC3339, *body.RestoreEnd)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "restoreEnd must be RFC3339.")
				return
			}
			restoreEnd = &t
		}
		id, err := backupservice.RecordRestoreDrill(r.Context(), d.Pool, backupservice.RecordRestoreDrillInput{
			DrillDate:          drillDate,
			BackupTimestamp:    backupTS,
			RestoreStart:       restoreStart,
			RestoreEnd:         restoreEnd,
			RPOAchievedMinutes: body.RPOAchievedMinutes,
			RTOAchievedMinutes: body.RTOAchievedMinutes,
			Pass:               body.Pass,
			SmokeTestOutput:    body.SmokeTestOutput,
			ConductedBy:        &conductorID,
			Notes:              body.Notes,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not record restore drill.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}
