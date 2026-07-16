package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func (d Deps) requireBoardManage(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID) bool {
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return false
	}
	return true
}

func boardMemberJSON(m board.BoardMember) map[string]any {
	return map[string]any{
		"boardId":   m.BoardID,
		"userId":    m.UserID,
		"role":      m.Role,
		"createdAt": m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func boardShareJSON(s board.BoardShare) map[string]any {
	out := map[string]any{
		"id":          s.ID,
		"boardId":     s.BoardID,
		"capability":  s.Capability,
		"hasPassword": s.HasPassword,
		"createdAt":   s.CreatedAt.UTC().Format(time.RFC3339),
	}
	if s.ExpiresAt != nil {
		out["expiresAt"] = s.ExpiresAt.UTC().Format(time.RFC3339)
	} else {
		out["expiresAt"] = nil
	}
	if s.RevokedAt != nil {
		out["revokedAt"] = s.RevokedAt.UTC().Format(time.RFC3339)
	} else {
		out["revokedAt"] = nil
	}
	if s.CreatedBy != nil {
		out["createdBy"] = *s.CreatedBy
	}
	return out
}

// handleListBoardMembers is GET .../boards/{board_id}/members
func (d Deps) handleListBoardMembers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireBoardManage(w, r, courseCode, viewer) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		if _, _, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID); !ok {
			return
		}
		members, err := board.ListMembers(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list members.")
			return
		}
		out := make([]map[string]any, 0, len(members))
		for _, m := range members {
			out = append(out, boardMemberJSON(m))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"members": out})
	}
}

// handleUpsertBoardMember is POST .../boards/{board_id}/members
func (d Deps) handleUpsertBoardMember() http.HandlerFunc {
	type reqBody struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireBoardManage(w, r, courseCode, viewer) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		if _, _, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID); !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		uid, err := uuid.Parse(strings.TrimSpace(in.UserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid userId.")
			return
		}
		role := in.Role
		if strings.TrimSpace(role) == "" {
			role = board.MemberRoleContributor
		}
		m, err := board.UpsertMember(r.Context(), d.Pool, courseCode, boardID, uid, role)
		if err != nil {
			if strings.Contains(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not add member.")
			return
		}
		if m == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		recordBoardAccessChange("board.member.upserted")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardMemberJSON(*m))
	}
}

// handleDeleteBoardMember is DELETE .../boards/{board_id}/members/{user_id}
func (d Deps) handleDeleteBoardMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireBoardManage(w, r, courseCode, viewer) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		userID := chi.URLParam(r, "user_id")
		if _, _, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID); !ok {
			return
		}
		okDel, err := board.RemoveMember(r.Context(), d.Pool, courseCode, boardID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not remove member.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Member not found.")
			return
		}
		recordBoardAccessChange("board.member.removed")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListBoardShares is GET .../boards/{board_id}/shares
func (d Deps) handleListBoardShares() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireBoardManage(w, r, courseCode, viewer) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		if _, _, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID); !ok {
			return
		}
		shares, err := board.ListShares(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list shares.")
			return
		}
		out := make([]map[string]any, 0, len(shares))
		for _, s := range shares {
			out = append(out, boardShareJSON(s))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"shares": out})
	}
}

// handleCreateBoardShare is POST .../boards/{board_id}/shares
func (d Deps) handleCreateBoardShare() http.HandlerFunc {
	type reqBody struct {
		Capability string  `json:"capability"`
		Password   string  `json:"password"`
		ExpiresAt  *string `json:"expiresAt"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireBoardManage(w, r, courseCode, viewer) {
			return
		}
		allowed, reason, okPol := d.externalSharingAllowedForCourse(w, r, courseCode)
		if !okPol {
			return
		}
		if !allowed {
			msg := "External board sharing is disabled."
			if reason == "minors_policy" {
				msg = "External board sharing is blocked for courses with minors."
			}
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, msg)
			return
		}
		boardID := chi.URLParam(r, "board_id")
		if _, _, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID); !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cap := in.Capability
		if strings.TrimSpace(cap) == "" {
			cap = board.ShareCapabilityView
		}
		var expires *time.Time
		if in.ExpiresAt != nil && strings.TrimSpace(*in.ExpiresAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.ExpiresAt))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "expiresAt must be RFC3339.")
				return
			}
			tt := t.UTC()
			expires = &tt
		}
		share, raw, err := board.CreateShare(r.Context(), d.Pool, courseCode, boardID, viewer, board.CreateShareInput{
			Capability: cap,
			Password:   in.Password,
			ExpiresAt:  expires,
		})
		if err != nil {
			if strings.Contains(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create share link.")
			return
		}
		if share == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.share.created")
		out := boardShareJSON(*share)
		out["token"] = raw
		out["url"] = "/board-links/" + raw
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleRevokeBoardShare is DELETE .../boards/{board_id}/shares/{share_id}
func (d Deps) handleRevokeBoardShare() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireBoardManage(w, r, courseCode, viewer) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		shareID := chi.URLParam(r, "share_id")
		if _, _, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID); !ok {
			return
		}
		okRev, err := board.RevokeShare(r.Context(), d.Pool, courseCode, boardID, shareID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not revoke share.")
			return
		}
		if !okRev {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Share not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.share.revoked")
		w.WriteHeader(http.StatusNoContent)
	}
}
