package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoDemo "github.com/lextures/lextures/server/internal/repos/demographics"
	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
	"github.com/lextures/lextures/server/internal/repos/organization"
	cfservice "github.com/lextures/lextures/server/internal/service/customfields"
)

func (d Deps) registerMeProfileDepthRoutes(r chi.Router) {
	r.Get("/api/v1/me/profile-fields", d.handleGetMyProfileFields())
	r.Patch("/api/v1/me/profile-fields", d.handlePatchMyProfileFields())
	r.Get("/api/v1/me/demographics", d.handleGetMyDemographics())
	r.Patch("/api/v1/me/demographics", d.handlePatchMyDemographics())
}

func studentVisibleDefinitions(defs []cfrepo.Definition) []map[string]any {
	out := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		if def.DeletedAt != nil {
			continue
		}
		if def.Visibility != cfrepo.VisibilityStudent {
			continue
		}
		out = append(out, definitionToJSON(def))
	}
	return out
}

func (d Deps) handleGetMyProfileFields() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.effectiveConfig().CustomFieldsEnabled {
			writeJSON(w, http.StatusOK, map[string]any{"fields": []any{}, "values": map[string]any{}})
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Organization not found.")
			return
		}
		svc := d.customFieldsService()
		defs, err := svc.ListDefinitions(r.Context(), orgID, cfrepo.EntityUser, false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile fields.")
			return
		}
		values, err := svc.GetUserValues(r.Context(), orgID, userID, cfservice.AudienceStudent, false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile field values.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"fields": studentVisibleDefinitions(defs),
			"values": values,
		})
	}
}

func (d Deps) handlePatchMyProfileFields() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.effectiveConfig().CustomFieldsEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom fields are not enabled.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Organization not found.")
			return
		}
		var body struct {
			Values       map[string]any `json:"values"`
			CustomFields map[string]any `json:"customFields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		incoming := body.Values
		if incoming == nil {
			incoming = body.CustomFields
		}
		if incoming == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "values object is required.")
			return
		}
		svc := d.customFieldsService()
		merged, valErrs, err := svc.SetUserValues(r.Context(), orgID, userID, incoming)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update profile fields.")
			return
		}
		if len(valErrs) > 0 {
			writeCustomFieldValidationErrors(w, valErrs)
			return
		}
		filtered, err := svc.GetUserValues(r.Context(), orgID, userID, cfservice.AudienceStudent, false)
		if err != nil {
			filtered = merged
		}
		writeJSON(w, http.StatusOK, map[string]any{"values": filtered})
	}
}

func (d Deps) handleGetMyDemographics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.demographicsEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := repoDemo.GetByStudentID(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load demographics.")
			return
		}
		if row == nil {
			writeJSON(w, http.StatusOK, map[string]any{"studentId": userID.String()})
			return
		}
		writeJSON(w, http.StatusOK, rowToJSON(row))
	}
}

func (d Deps) handlePatchMyDemographics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.demographicsEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			FreeLunch         *bool   `json:"freeLunch"`
			ReducedLunch      *bool   `json:"reducedLunch"`
			EllStatus         *bool   `json:"ellStatus"`
			DisabilityStatus  *bool   `json:"disabilityStatus"`
			RaceEthnicityCode *string `json:"raceEthnicityCode"`
			HomelessIndicator *bool   `json:"homelessIndicator"`
			MigrantIndicator  *bool   `json:"migrantIndicator"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		row, err := repoDemo.Upsert(r.Context(), d.Pool, userID, repoDemo.UpsertInput{
			FreeLunch:         body.FreeLunch,
			ReducedLunch:      body.ReducedLunch,
			EllStatus:         body.EllStatus,
			DisabilityStatus:  body.DisabilityStatus,
			RaceEthnicityCode: body.RaceEthnicityCode,
			HomelessIndicator: body.HomelessIndicator,
			MigrantIndicator:  body.MigrantIndicator,
			DataSource:        "self_reported",
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update demographics.")
			return
		}
		writeJSON(w, http.StatusOK, rowToJSON(row))
	}
}