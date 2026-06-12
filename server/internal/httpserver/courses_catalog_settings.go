package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
)

type catalogSettingsResponse struct {
	View                 string            `json:"view"`
	KanbanColumnLabels   map[string]string `json:"kanbanColumnLabels"`
	HiddenColumnExpanded bool              `json:"hiddenColumnExpanded"`
	Nicknames            map[string]string `json:"nicknames"`
}

type putCatalogSettingsBody struct {
	View                 *string           `json:"view"`
	KanbanColumnLabels   map[string]string `json:"kanbanColumnLabels"`
	HiddenColumnExpanded *bool             `json:"hiddenColumnExpanded"`
}

type putCatalogNicknameBody struct {
	CourseID string  `json:"courseId"`
	Nickname *string `json:"nickname"`
}

type putKanbanBoardBody struct {
	Columns map[string][]string `json:"columns"`
}

type putCatalogPinBody struct {
	CourseID string `json:"courseId"`
	Pinned   bool   `json:"pinned"`
}

type pinnedCoursesResponse struct {
	Courses []course.PinnedCourseSummary `json:"courses"`
}

func (d Deps) handleGetCourseCatalogSettings() http.HandlerFunc {
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
		ctx := r.Context()
		prefs, err := course.GetUserCatalogPrefs(ctx, d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load catalog settings.")
			return
		}
		nicknames, err := course.ListUserCatalogNicknames(ctx, d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load catalog settings.")
			return
		}
		if nicknames == nil {
			nicknames = map[string]string{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(catalogSettingsResponse{
			View:                 prefs.ViewType,
			KanbanColumnLabels:   prefs.KanbanColumnLabels,
			HiddenColumnExpanded: prefs.HiddenColumnExpanded,
			Nicknames:            nicknames,
		})
	}
}

func (d Deps) handlePutCourseCatalogSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body putCatalogSettingsBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		patch := course.UserCatalogPrefs{}
		hasView := body.View != nil
		hasLabels := body.KanbanColumnLabels != nil
		hasHidden := body.HiddenColumnExpanded != nil
		if hasView {
			patch.ViewType = strings.TrimSpace(*body.View)
		}
		if hasLabels {
			patch.KanbanColumnLabels = body.KanbanColumnLabels
		}
		if hasHidden {
			patch.HiddenColumnExpanded = *body.HiddenColumnExpanded
		}
		if !hasView && !hasLabels && !hasHidden {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No settings provided.")
			return
		}
		updated, err := course.UpsertUserCatalogPrefs(r.Context(), d.Pool, userID, patch, hasView, hasLabels, hasHidden)
		if err != nil {
			if strings.Contains(err.Error(), "invalid view") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid view type.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save catalog settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(catalogSettingsResponse{
			View:                 updated.ViewType,
			KanbanColumnLabels:   updated.KanbanColumnLabels,
			HiddenColumnExpanded: updated.HiddenColumnExpanded,
			Nicknames:            map[string]string{},
		})
	}
}

func (d Deps) handlePutCourseCatalogNickname() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body putCatalogNicknameBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(body.CourseID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
			return
		}
		if err := course.UpsertUserCatalogNickname(r.Context(), d.Pool, userID, courseID, body.Nickname); err != nil {
			if strings.Contains(err.Error(), "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Nickname is too long.")
				return
			}
			if strings.Contains(err.Error(), "not in your catalog") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save nickname.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handlePutCourseCatalogOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			CourseIDs []uuid.UUID `json:"courseIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := course.ReplaceUserCatalogOrder(r.Context(), d.Pool, userID, body.CourseIDs); err != nil {
			if strings.Contains(err.Error(), "not in your catalog") || strings.Contains(err.Error(), "duplicate") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save catalog order.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handlePutCourseKanbanBoard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body putKanbanBoardBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Columns == nil {
			body.Columns = map[string][]string{}
		}
		columns := map[string][]uuid.UUID{}
		for col, ids := range body.Columns {
			for _, rawID := range ids {
				id, err := uuid.Parse(strings.TrimSpace(rawID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id in kanban board.")
					return
				}
				columns[col] = append(columns[col], id)
			}
		}
		if err := course.ReplaceUserKanbanBoard(r.Context(), d.Pool, userID, columns); err != nil {
			if strings.Contains(err.Error(), "invalid kanban") || strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "not in your catalog") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save kanban board.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleGetCourseCatalogPins() http.HandlerFunc {
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
		courses, err := course.ListUserPinnedCourseSummaries(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load pinned courses.")
			return
		}
		if courses == nil {
			courses = []course.PinnedCourseSummary{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(pinnedCoursesResponse{Courses: courses})
	}
}

func (d Deps) handlePutCourseCatalogPin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body putCatalogPinBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(body.CourseID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
			return
		}
		if err := course.SetUserCatalogPin(r.Context(), d.Pool, userID, courseID, body.Pinned); err != nil {
			if strings.Contains(err.Error(), "not in your catalog") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			if strings.Contains(err.Error(), "pin limit") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "You can pin at most 20 courses.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save pin.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleMigrateCourseCatalogLocalStorage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body putCatalogSettingsBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		patch := course.UserCatalogPrefs{}
		hasView := body.View != nil
		hasHidden := body.HiddenColumnExpanded != nil
		if hasView {
			patch.ViewType = strings.TrimSpace(*body.View)
		}
		if hasHidden {
			patch.HiddenColumnExpanded = *body.HiddenColumnExpanded
		}
		if !hasView && !hasHidden {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, err := course.UpsertUserCatalogPrefs(r.Context(), d.Pool, userID, patch, hasView, false, hasHidden)
		if err != nil {
			if strings.Contains(err.Error(), "invalid view") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid view type.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to migrate catalog settings.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
