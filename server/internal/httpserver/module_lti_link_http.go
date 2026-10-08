package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	ltidb "github.com/lextures/lextures/server/internal/repos/lti"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// moduleLTILinkResponse is GET /api/v1/courses/{course_code}/lti-links/{item_id}.
type moduleLTILinkResponse struct {
	ItemID           uuid.UUID `json:"itemId"`
	Title            string    `json:"title"`
	ExternalToolID   uuid.UUID `json:"externalToolId"`
	ExternalToolName string    `json:"externalToolName"`
	ResourceLinkID   string    `json:"resourceLinkId"`
	LineItemURL      *string   `json:"lineItemUrl"`
}

// loadedModuleLTILink is an lti_link module item the viewer may open, with its tool.
type loadedModuleLTILink struct {
	courseID uuid.UUID
	itemID   uuid.UUID
	viewer   uuid.UUID
	title    string
	link     *ltidb.ResourceLink
	tool     *ltidb.ExternalTool
}

// loadModuleLTILinkForViewer resolves an lti_link module item for the current user. Staff with
// item:create on the course see any link; everyone else only links visible to students.
// It writes the error response and returns false when the item can't be opened.
func (d Deps) loadModuleLTILinkForViewer(w http.ResponseWriter, r *http.Request) (loadedModuleLTILink, bool) {
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return loadedModuleLTILink{}, false
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return loadedModuleLTILink{}, false
	}
	ctx := r.Context()
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return loadedModuleLTILink{}, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return loadedModuleLTILink{}, false
	}
	canEdit, err := rbac.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return loadedModuleLTILink{}, false
	}
	if !canEdit {
		visible, err := coursestructure.LTILinkVisibleToStudent(ctx, d.Pool, *cid, itemID, viewer, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check LTI link access.")
			return loadedModuleLTILink{}, false
		}
		if !visible {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return loadedModuleLTILink{}, false
		}
	}
	item, err := coursestructure.GetItemRow(ctx, d.Pool, *cid, itemID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load LTI link.")
		return loadedModuleLTILink{}, false
	}
	if item == nil || item.Kind != "lti_link" {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return loadedModuleLTILink{}, false
	}
	link, err := ltidb.GetResourceLinkForStructureItem(ctx, d.Pool, *cid, itemID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load LTI link.")
		return loadedModuleLTILink{}, false
	}
	if link == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return loadedModuleLTILink{}, false
	}
	tool, err := ltidb.GetExternalToolByID(ctx, d.Pool, link.ExternalToolID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load external tool.")
		return loadedModuleLTILink{}, false
	}
	if tool == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "External tool not found.")
		return loadedModuleLTILink{}, false
	}
	title := item.Title
	if title == "" && link.Title != nil {
		title = *link.Title
	}
	return loadedModuleLTILink{
		courseID: *cid, itemID: itemID, viewer: viewer, title: title, link: link, tool: tool,
	}, true
}

// handleGetModuleLTILink is GET /api/v1/courses/{course_code}/lti-links/{item_id}.
func (d Deps) handleGetModuleLTILink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, ok := d.loadModuleLTILinkForViewer(w, r)
		if !ok {
			return
		}
		out := moduleLTILinkResponse{
			ItemID:           l.itemID,
			Title:            l.title,
			ExternalToolID:   l.link.ExternalToolID,
			ExternalToolName: l.tool.Name,
			ResourceLinkID:   l.link.ResourceLinkID,
			LineItemURL:      l.link.LineItemURL,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePostModuleLTIEmbedTicket is POST /api/v1/courses/{course_code}/lti-links/{item_id}/embed-ticket.
// It returns a short-lived ticket for GET /api/v1/lti/consumer/frame, which can't send a Bearer header.
func (d Deps) handlePostModuleLTIEmbedTicket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLtiHandler(w) {
			return
		}
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		l, ok := d.loadModuleLTILinkForViewer(w, r)
		if !ok {
			return
		}
		if !l.tool.Active {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "External tool is not active.")
			return
		}
		ticket, err := d.JWTSigner.SignLTIEmbedTicket(l.viewer.String(), l.courseID.String(), l.itemID.String())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create launch ticket.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"ticket": ticket})
	}
}
