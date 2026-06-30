package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	licenserepo "github.com/lextures/lextures/server/internal/repos/license"
	"github.com/lextures/lextures/server/internal/service/licensesvc"
)

func (d Deps) licenseOverviewJSON(r *http.Request, orgID uuid.UUID) any {
	if !d.effectiveConfig().SeatManagementEnabled {
		return nil
	}
	lic, err := licenserepo.Effective(r.Context(), d.Pool, orgID)
	if err != nil {
		return nil
	}
	return licenseToJSON(lic)
}

func (d Deps) seatManagementEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().SeatManagementEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Seat management is not enabled.")
		return false
	}
	return true
}

func licenseToJSON(r licenserepo.Row) map[string]any {
	out := map[string]any{
		"orgId":      r.OrgID.String(),
		"tier":       r.Tier,
		"maxSeats":   r.MaxSeats,
		"usedSeats":  r.UsedSeats,
		"unlimited":  r.MaxSeats < 0,
		"notes":      r.Notes,
		"createdAt":  r.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":  r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if r.ID != uuid.Nil {
		out["id"] = r.ID.String()
	}
	if r.OrgName != "" {
		out["orgName"] = r.OrgName
	}
	if r.OrgSlug != "" {
		out["orgSlug"] = r.OrgSlug
	}
	if r.ContractStart != nil {
		out["contractStart"] = r.ContractStart.Format("2006-01-02")
	}
	if r.ContractEnd != nil {
		out["contractEnd"] = r.ContractEnd.Format("2006-01-02")
	}
	if r.MaxSeats > 0 {
		out["percentUsed"] = licensesvc.UtilizationPercent(r.UsedSeats, r.MaxSeats)
	}
	if licensesvc.ContractExpiringSoon(r, 30) {
		out["contractExpiringSoon"] = true
	}
	return out
}

func writeSeatLimitError(w http.ResponseWriter) {
	apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeSeatLimitReached,
		"Your organization has reached its licensed seat limit. Contact your administrator to request additional seats.")
}

func (d Deps) licenseService() *licensesvc.Service {
	return licensesvc.New(d.Pool, d.effectiveConfig())
}

func (d Deps) registerAdminLicenseRoutes(r chi.Router) {
	r.Get("/api/v1/admin-console/license", d.handleAdminConsoleLicenseGet())
	r.Get("/api/v1/admin/licenses", d.handleAdminLicensesList())
	r.Patch("/api/v1/admin/licenses/{orgId}", d.handleAdminLicensePatch())
	r.Post("/api/v1/admin/licenses/{orgId}/resync", d.handleAdminLicenseResync())
}

func (d Deps) handleAdminConsoleLicenseGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.seatManagementEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		lic, err := licenserepo.Effective(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load license.")
			return
		}
		writeJSON(w, http.StatusOK, licenseToJSON(lic))
	}
}

func (d Deps) handleAdminLicensesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.seatManagementEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		limit := 100
		offset := 0
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		rows, err := licenserepo.List(r.Context(), d.Pool, licenserepo.ListParams{Limit: limit, Offset: offset})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list licenses.")
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, licenseToJSON(row))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
	}
}

type patchLicenseBody struct {
	Tier          *string `json:"tier"`
	MaxSeats      *int    `json:"maxSeats"`
	ContractStart *string `json:"contractStart"`
	ContractEnd   *string `json:"contractEnd"`
	Notes         *string `json:"notes"`
}

func parseLicenseDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	trim := strings.TrimSpace(*s)
	if trim == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", trim)
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}

func (d Deps) handleAdminLicensePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.seatManagementEnabled(w) {
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body patchLicenseBody
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		start, err := parseLicenseDate(body.ContractStart)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid contractStart; use YYYY-MM-DD.")
			return
		}
		end, err := parseLicenseDate(body.ContractEnd)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid contractEnd; use YYYY-MM-DD.")
			return
		}
		updated, err := licenserepo.Upsert(r.Context(), d.Pool, orgID, licenserepo.Patch{
			Tier:          body.Tier,
			MaxSeats:      body.MaxSeats,
			ContractStart: start,
			ContractEnd:   end,
			Notes:         body.Notes,
			UpdatedBy:     &actor,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		_ = d.licenseService().MaybeSendUtilizationAlerts(r.Context(), orgID)
		writeJSON(w, http.StatusOK, licenseToJSON(updated))
	}
}

func (d Deps) handleAdminLicenseResync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.seatManagementEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if err := licenserepo.RefreshUsedSeats(r.Context(), d.Pool, orgID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resync seat counter.")
			return
		}
		lic, err := licenserepo.Effective(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load license.")
			return
		}
		writeJSON(w, http.StatusOK, licenseToJSON(lic))
	}
}
