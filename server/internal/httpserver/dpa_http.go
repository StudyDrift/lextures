package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repodpa "github.com/lextures/lextures/server/internal/repos/dpa"
	"github.com/lextures/lextures/server/internal/repos/organization"
	dpaservice "github.com/lextures/lextures/server/internal/service/dpa"
)

func (d Deps) dpaEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().DPAPortalEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "DPA portal is not enabled.")
		return false
	}
	return true
}

// requireDPAOrgAdmin authenticates the caller and resolves their org ID.
func (d Deps) requireDPAOrgAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, orgID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	oid, err := organization.OrgIDForUser(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return uid, oid, true
}

// requireDPAAdmin additionally enforces compliance:dpa:admin permission.
func (d Deps) requireDPAAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := dpaservice.CheckAdmin(r.Context(), d.Pool, uid)
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

func (d Deps) registerDPARoutes(r chi.Router) {
	r.Get("/api/v1/compliance/dpa/current", d.handleGetDPACurrent())
	r.Post("/api/v1/compliance/dpa/accept", d.handlePostDPAAccept())
	r.Get("/api/v1/compliance/dpa/acceptances", d.handleGetDPAAcceptances())
	r.Get("/api/v1/compliance/data-inventory", d.handleGetDataInventory())
	r.Get("/api/v1/compliance/data-inventory/export.csv", d.handleGetDataInventoryCSV())
}

// GET /api/v1/compliance/dpa/current
func (d Deps) handleGetDPACurrent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dpaEnabled(w) {
			return
		}
		userID, orgID, ok := d.requireDPAOrgAdmin(w, r)
		_ = userID
		if !ok {
			return
		}
		v, err := dpaservice.GetCurrentVersion(r.Context(), d.Pool)
		if err != nil {
			if errors.Is(err, dpaservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No DPA version is currently active.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DPA version.")
			return
		}
		signed, err := dpaservice.GetOrgAcceptance(r.Context(), d.Pool, orgID, v.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not check DPA status.")
			return
		}
		items, err := dpaservice.ListDataInventory(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load data inventory.")
			return
		}
		tpl := dpaservice.GenerateNDPATemplate(v, items)
		resp := map[string]any{
			"version":  dpaVersionJSON(v),
			"signed":   signed != nil,
			"template": tpl,
		}
		if signed != nil {
			resp["acceptedAt"] = signed.AcceptedAt.UTC().Format(time.RFC3339)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// POST /api/v1/compliance/dpa/accept
func (d Deps) handlePostDPAAccept() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dpaEnabled(w) {
			return
		}
		userID, orgID, ok := d.requireDPAOrgAdmin(w, r)
		if !ok {
			return
		}
		v, err := dpaservice.GetCurrentVersion(r.Context(), d.Pool)
		if err != nil {
			if errors.Is(err, dpaservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No DPA version is currently active.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DPA version.")
			return
		}
		id, err := dpaservice.AcceptDPA(r.Context(), d.Pool, orgID, v.ID, userID, r.RemoteAddr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not record DPA acceptance.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String(), "versionStr": v.VersionStr})
	}
}

// GET /api/v1/compliance/dpa/acceptances
func (d Deps) handleGetDPAAcceptances() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dpaEnabled(w) {
			return
		}
		_, ok := d.requireDPAAdmin(w, r)
		if !ok {
			return
		}
		acceptances, err := dpaservice.ListAcceptances(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DPA acceptances.")
			return
		}
		out := make([]map[string]any, 0, len(acceptances))
		for _, a := range acceptances {
			m := map[string]any{
				"id":           a.ID.String(),
				"orgId":        a.OrgID.String(),
				"dpaVersionId": a.DPAVersionID.String(),
				"acceptedBy":   a.AcceptedBy.String(),
				"acceptedAt":   a.AcceptedAt.UTC().Format(time.RFC3339),
			}
			if a.IPAddress != nil {
				m["ipAddress"] = *a.IPAddress
			}
			out = append(out, m)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"acceptances": out})
	}
}

// GET /api/v1/compliance/data-inventory
func (d Deps) handleGetDataInventory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dpaEnabled(w) {
			return
		}
		_, _, ok := d.requireDPAOrgAdmin(w, r)
		if !ok {
			return
		}
		items, err := dpaservice.ListDataInventory(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load data inventory.")
			return
		}
		type item struct {
			ID                      string   `json:"id"`
			ElementName             string   `json:"elementName"`
			Category                string   `json:"category"`
			Purpose                 string   `json:"purpose"`
			LegalBasis              string   `json:"legalBasis"`
			RetentionDays           *int     `json:"retentionDays,omitempty"`
			SharedWithSubProcessors bool     `json:"sharedWithSubProcessors"`
			SubProcessorNames       []string `json:"subProcessorNames"`
			UpdatedAt               string   `json:"updatedAt"`
		}
		out := make([]item, 0, len(items))
		for _, i := range items {
			out = append(out, item{
				ID:                      i.ID.String(),
				ElementName:             i.ElementName,
				Category:                i.Category,
				Purpose:                 i.Purpose,
				LegalBasis:              i.LegalBasis,
				RetentionDays:           i.RetentionDays,
				SharedWithSubProcessors: i.SharedWithSubProcessors,
				SubProcessorNames:       dpaSlice(i.SubProcessorNames),
				UpdatedAt:               i.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}
}

// GET /api/v1/compliance/data-inventory/export.csv
func (d Deps) handleGetDataInventoryCSV() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dpaEnabled(w) {
			return
		}
		_, _, ok := d.requireDPAOrgAdmin(w, r)
		if !ok {
			return
		}
		csvBytes, err := dpaservice.SDPCCSVExport(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not generate CSV export.")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="sdpc-data-inventory.csv"`)
		_, _ = w.Write(csvBytes)
	}
}

func dpaVersionJSON(v *repodpa.DPAVersion) map[string]any {
	m := map[string]any{
		"id":          v.ID.String(),
		"versionStr":  v.VersionStr,
		"templateUrl": v.TemplateURL,
		"effectiveAt": v.EffectiveAt.UTC().Format(time.RFC3339),
	}
	if v.Notes != nil {
		m["notes"] = *v.Notes
	}
	return m
}

func dpaSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
