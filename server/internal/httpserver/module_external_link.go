package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleexternallinks"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

type moduleExternalLinkResponse struct {
	ItemID          uuid.UUID  `json:"itemId"`
	Title           string     `json:"title"`
	URL             string     `json:"url"`
	Provider        string     `json:"provider"`
	ExternalID      *string    `json:"externalId"`
	IconURL         *string    `json:"iconUrl"`
	LicenseSPDX     *string    `json:"licenseSpdx"`
	AttributionText *string    `json:"attributionText"`
	OERProvider     *string    `json:"oerProvider"`
	UpdatedAt       *time.Time `json:"updatedAt"`
}

// handleGetModuleExternalLink is GET /api/v1/courses/{course_code}/external-links/{item_id}.
func (d Deps) handleGetModuleExternalLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			visible, err := coursestructure.ExternalLinkVisibleToStudent(
				r.Context(), d.Pool, *cid, itemID, viewer, time.Now().UTC(),
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check external link access.")
				return
			}
			if !visible {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
				return
			}
		}
		title, url, provider, externalID, iconURL, licenseSPDX, attributionText, oerProvider, updatedAt, err :=
			coursemoduleexternallinks.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load external link.")
			return
		}
		if title == "" && url == "" && updatedAt == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if provider == "" {
			provider = "url"
		}
		out := moduleExternalLinkResponse{
			ItemID:          itemID,
			Title:           title,
			URL:             url,
			Provider:        provider,
			ExternalID:      externalID,
			IconURL:         iconURL,
			LicenseSPDX:     licenseSPDX,
			AttributionText: attributionText,
			OERProvider:     oerProvider,
			UpdatedAt:       updatedAt,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

type patchExternalLinkBody struct {
	URL        string  `json:"url"`
	Provider   string  `json:"provider"`
	ExternalID *string `json:"externalId"`
	IconURL    *string `json:"iconUrl"`
}

// handlePatchModuleExternalLink is PATCH /api/v1/courses/{course_code}/external-links/{item_id}.
func (d Deps) handlePatchModuleExternalLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var req patchExternalLinkBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cleanURL := strings.TrimSpace(req.URL)
		if cleanURL != "" {
			_, err := coursemoduleexternallinks.ValidateExternalHTTPURL(cleanURL)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
		}
		provider := strings.TrimSpace(req.Provider)
		if provider == "" {
			provider = "url"
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		updatedAt, err := coursemoduleexternallinks.UpdateLink(r.Context(), d.Pool, *cid, itemID, cleanURL, provider, req.ExternalID, req.IconURL)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update external link.")
			return
		}
		if updatedAt == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		title, _, _, _, _, _, _, _, _, err := coursemoduleexternallinks.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload external link.")
			return
		}
		out := moduleExternalLinkResponse{
			ItemID:     itemID,
			Title:      title,
			URL:        cleanURL,
			Provider:   provider,
			ExternalID: req.ExternalID,
			IconURL:    req.IconURL,
			UpdatedAt:  updatedAt,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
