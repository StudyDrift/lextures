package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func boardSectionJSON(s board.Section) map[string]any {
	return map[string]any{
		"id":        s.ID,
		"boardId":   s.BoardID,
		"title":     s.Title,
		"sortIndex": s.SortIndex,
		"createdAt": s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// handleListBoardSections is GET .../boards/{board_id}/sections
func (d Deps) handleListBoardSections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load board.")
			return
		}
		if b == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		sections, err := board.ListSections(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list sections.")
			return
		}
		out := make([]map[string]any, 0, len(sections))
		for _, s := range sections {
			out = append(out, boardSectionJSON(s))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"sections": out})
	}
}

// handleCreateBoardSection is POST .../boards/{board_id}/sections
func (d Deps) handleCreateBoardSection() http.HandlerFunc {
	type reqBody struct {
		Title     string   `json:"title"`
		SortIndex *float64 `json:"sortIndex"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load board.")
			return
		}
		if b == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		created, err := board.CreateSection(r.Context(), d.Pool, courseCode, boardID, in.Title, in.SortIndex)
		if err != nil {
			if strings.Contains(err.Error(), "title") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create section.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.section.created")
		notifyBoardPeers(r.Context(), boardID, "section.created", "")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardSectionJSON(*created))
	}
}

// handlePatchBoardSection is PATCH .../boards/{board_id}/sections/{section_id}
func (d Deps) handlePatchBoardSection() http.HandlerFunc {
	type reqBody struct {
		Title     *string  `json:"title"`
		SortIndex *float64 `json:"sortIndex"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		boardID := chi.URLParam(r, "board_id")
		sectionID := chi.URLParam(r, "section_id")
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		updated, err := board.PatchSection(r.Context(), d.Pool, courseCode, boardID, sectionID, board.PatchSectionInput{
			Title:     in.Title,
			SortIndex: in.SortIndex,
		})
		if err != nil {
			if strings.Contains(err.Error(), "title") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update section.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Section not found.")
			return
		}
		notifyBoardPeers(r.Context(), boardID, "section.updated", "")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardSectionJSON(*updated))
	}
}

// handleDeleteBoardSection is DELETE .../boards/{board_id}/sections/{section_id}
func (d Deps) handleDeleteBoardSection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		boardID := chi.URLParam(r, "board_id")
		sectionID := chi.URLParam(r, "section_id")
		okDel, err := board.DeleteSection(r.Context(), d.Pool, courseCode, boardID, sectionID)
		if err != nil {
			if strings.Contains(err.Error(), "cannot delete") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete section.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Section not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.section.deleted")
		notifyBoardPeers(r.Context(), boardID, "section.deleted", "")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleArrangeBoardPost is PATCH .../boards/{board_id}/posts/{post_id}/arrange
func (d Deps) handleArrangeBoardPost() http.HandlerFunc {
	type reqBody struct {
		SectionID *string             `json:"sectionId"`
		SortIndex *float64            `json:"sortIndex"`
		Position  *board.PostPosition `json:"position"`
		EventDate *string             `json:"eventDate"`
		Lat       *float64            `json:"lat"`
		Lng       *float64            `json:"lng"`
		ClearGeo  *bool               `json:"clearGeo"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")

		b, caps, okAccess := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !okAccess {
			return
		}
		existing, err := board.GetPost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load post.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}

		if !caps.CanArrange && !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to rearrange this card.")
			return
		}
		if d.writeGateReject(w, board.CheckWriteAllowed(b, caps.CanManage, board.WriteArrange, time.Now().UTC())) {
			return
		}
		if err := board.CanArrangePost(b.LayoutLocked, caps.CanManage, existing.AuthorID, viewer); err != nil {
			if errors.Is(err, board.ErrLayoutLocked) || errors.Is(err, board.ErrArrangeForbidden) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to rearrange this card.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}

		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		arrange := board.ArrangePostInput{
			SectionID: in.SectionID,
			SortIndex: in.SortIndex,
			Position:  in.Position,
			Lat:       in.Lat,
			Lng:       in.Lng,
		}
		if in.ClearGeo != nil && *in.ClearGeo {
			arrange.ClearGeo = true
		}
		if in.EventDate != nil {
			raw := strings.TrimSpace(*in.EventDate)
			if raw == "" {
				arrange.ClearEventDate = true
			} else {
				t, err := time.Parse(time.RFC3339, raw)
				if err != nil {
					t, err = time.Parse("2006-01-02", raw)
					if err != nil {
						apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "eventDate must be RFC3339 or YYYY-MM-DD.")
						return
					}
				}
				arrange.EventDate = &t
			}
		}

		updated, err := board.ArrangePost(r.Context(), d.Pool, courseCode, boardID, postID, arrange)
		if err != nil {
			if strings.Contains(err.Error(), "section not found") ||
				strings.Contains(err.Error(), "lat") ||
				strings.Contains(err.Error(), "lng") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not arrange post.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.post.arranged")
		notifyBoardPeers(r.Context(), boardID, "post.arranged", postID)
		avOn := d.effectiveConfig().AvScanningEnabled
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardPostJSON(*updated, courseCode, avOn))
	}
}
