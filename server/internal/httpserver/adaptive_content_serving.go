package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

func (d Deps) registerAdaptiveContentServingRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/adaptive-content/optout", d.handleAdaptiveContentOptoutGet())
	r.Put("/api/v1/courses/{course_code}/adaptive-content/optout", d.handleAdaptiveContentOptoutPut())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/viewed-original", d.handleAdaptiveContentViewedOriginal())
}

// handleAdaptiveContentOptoutGet is GET .../adaptive-content/optout (student).
func (d Deps) handleAdaptiveContentOptoutGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
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

		optoutAllowed := true
		if settings, err := acrepo.GetSettings(r.Context(), d.Pool, *cid); err == nil && settings != nil {
			optoutAllowed = settings.StudentOptoutAllowed
		} else if err == nil {
			optoutAllowed = acrepo.DefaultSettings(*cid).StudentOptoutAllowed
		}

		optedOut, err := acrepo.IsOptedOut(r.Context(), d.Pool, *cid, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load opt-out preference.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.OptoutResponse{
			OptedOut:      optedOut,
			OptoutAllowed: optoutAllowed,
		})
	}
}

// handleAdaptiveContentOptoutPut is PUT .../adaptive-content/optout (student).
func (d Deps) handleAdaptiveContentOptoutPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
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

		settings, err := acrepo.GetSettings(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		var optoutAllowed bool
		if settings != nil {
			optoutAllowed = settings.StudentOptoutAllowed
		} else {
			optoutAllowed = acrepo.DefaultSettings(*cid).StudentOptoutAllowed
		}
		if !optoutAllowed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This course does not allow opting out of adaptive content.")
			return
		}

		var body acmodel.OptoutPutRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}

		row, err := acrepo.SetOptout(r.Context(), d.Pool, *cid, viewer, body.OptedOut)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save opt-out preference.")
			return
		}

		// Audit.
		subj := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, nil, &subj, &subj, acsvc.EventOptoutChanged, map[string]any{
			"optedOut": row.OptedOut,
		})
		if row.OptedOut {
			acsvc.IncOptout()
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.OptoutResponse{
			OptedOut:      row.OptedOut,
			OptoutAllowed: optoutAllowed,
		})
	}
}

// handleAdaptiveContentViewedOriginal is POST .../units/{unit_id}/viewed-original (student).
func (d Deps) handleAdaptiveContentViewedOriginal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
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
		unit, err := acrepo.GetUnit(r.Context(), d.Pool, *cid, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load unit.")
			return
		}
		if unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}

		clicks, err := acsvc.RecordViewedOriginal(r.Context(), d.Pool, *cid, unitID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record view.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.ViewedOriginalResponse{ViewOriginalClicks: clicks})
	}
}
