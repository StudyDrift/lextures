package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// handleGetBoardModerationQueue is GET .../boards/{board_id}/moderation/queue
func (d Deps) handleGetBoardModerationQueue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to moderate this board.")
			return
		}
		pending, err := board.ListPendingPosts(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load moderation queue.")
			return
		}
		reports, err := board.ListOpenReports(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reports.")
			return
		}
		avOn := d.effectiveConfig().AvScanningEnabled
		pendingOut := make([]map[string]any, 0, len(pending))
		for _, p := range pending {
			pendingOut = append(pendingOut, boardPostJSONWithAttribution(p, courseCode, avOn, b.Attribution, caps))
		}
		userReports := make([]map[string]any, 0)
		flagged := make([]map[string]any, 0)
		for _, rep := range reports {
			row := reportJSON(rep)
			switch rep.Kind {
			case board.ReportKindFilter, board.ReportKindAVBlocked:
				flagged = append(flagged, row)
			default:
				userReports = append(userReports, row)
			}
		}
		_ = b
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pending":  pendingOut,
			"reports":  userReports,
			"flagged":  flagged,
			"minorsFloor": d.minorsModerationFloor(r.Context(), courseCode),
		})
	}
}

// handleApproveBoardPost is POST .../posts/{post_id}/approve
func (d Deps) handleApproveBoardPost() http.HandlerFunc {
	return d.handlePostModerationAction(board.PostStatusApproved, board.ModActionApprove)
}

// handleRejectBoardPost is POST .../posts/{post_id}/reject
func (d Deps) handleRejectBoardPost() http.HandlerFunc {
	return d.handlePostModerationAction(board.PostStatusRejected, board.ModActionReject)
}

func (d Deps) handlePostModerationAction(status, action string) http.HandlerFunc {
	type reqBody struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to moderate this board.")
			return
		}
		var in reqBody
		_ = json.NewDecoder(r.Body).Decode(&in)
		updated, err := board.SetPostStatus(r.Context(), d.Pool, courseCode, boardID, postID, status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update post.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		tid, _ := uuid.Parse(postID)
		_ = board.InsertModerationLog(r.Context(), d.Pool, boardID, &viewer, action, board.TargetPost, &tid, in.Reason)
		telemetry.RecordBusinessEvent("board.moderation." + action)
		notifyBoardPeers(r.Context(), boardID, "post.moderated", postID)
		_ = b
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardPostJSONWithAttribution(*updated, courseCode, d.effectiveConfig().AvScanningEnabled, b.Attribution, caps))
	}
}

// handleHideBoardPost is POST .../posts/{post_id}/hide
func (d Deps) handleHideBoardPost() http.HandlerFunc {
	return d.handleHideRemovePost(false)
}

// handleRemoveBoardPost is POST .../posts/{post_id}/remove
func (d Deps) handleRemoveBoardPost() http.HandlerFunc {
	return d.handleHideRemovePost(true)
}

func (d Deps) handleHideRemovePost(remove bool) http.HandlerFunc {
	type reqBody struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to moderate this board.")
			return
		}
		var in reqBody
		_ = json.NewDecoder(r.Body).Decode(&in)
		var updated *board.Post
		var err error
		action := board.ModActionHide
		if remove {
			action = board.ModActionRemove
			updated, err = board.SoftRemovePost(r.Context(), d.Pool, courseCode, boardID, postID)
		} else {
			updated, err = board.SetPostHidden(r.Context(), d.Pool, courseCode, boardID, postID, true)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update post.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		tid, _ := uuid.Parse(postID)
		_ = board.InsertModerationLog(r.Context(), d.Pool, boardID, &viewer, action, board.TargetPost, &tid, in.Reason)
		telemetry.RecordBusinessEvent("board.moderation." + action)
		notifyBoardPeers(r.Context(), boardID, "post.moderated", postID)
		_ = b
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardPostJSONWithAttribution(*updated, courseCode, d.effectiveConfig().AvScanningEnabled, b.Attribution, caps))
	}
}

// handleHideBoardComment is POST .../comments/{comment_id}/hide
func (d Deps) handleHideBoardComment() http.HandlerFunc {
	return d.handleHideRemoveComment(false)
}

// handleRemoveBoardComment is POST .../comments/{comment_id}/remove
func (d Deps) handleRemoveBoardComment() http.HandlerFunc {
	return d.handleHideRemoveComment(true)
}

func (d Deps) handleHideRemoveComment(remove bool) http.HandlerFunc {
	type reqBody struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		commentID := chi.URLParam(r, "comment_id")
		_, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to moderate this board.")
			return
		}
		var in reqBody
		_ = json.NewDecoder(r.Body).Decode(&in)
		updated, err := board.SoftHideComment(r.Context(), d.Pool, courseCode, boardID, postID, commentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update comment.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Comment not found.")
			return
		}
		action := board.ModActionHide
		if remove {
			action = board.ModActionRemove
		}
		tid, _ := uuid.Parse(commentID)
		_ = board.InsertModerationLog(r.Context(), d.Pool, boardID, &viewer, action, board.TargetComment, &tid, in.Reason)
		telemetry.RecordBusinessEvent("board.moderation.comment_" + action)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardCommentJSONWithAttribution(*updated, board.AttributionNamed, caps))
	}
}

// handleCreateBoardReport is POST .../boards/{board_id}/reports
func (d Deps) handleCreateBoardReport() http.HandlerFunc {
	type reqBody struct {
		PostID    *string `json:"postId"`
		CommentID *string `json:"commentId"`
		Reason    string  `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanView {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have access to this board.")
			return
		}
		n, err := board.CountRecentReportsByReporter(r.Context(), d.Pool, boardID, viewer, time.Now().UTC().Add(-boardReportRateWindow))
		if err == nil && n >= boardReportRateLimit {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many reports. Please try again later.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		created, err := board.CreateReport(r.Context(), d.Pool, courseCode, boardID, &viewer, in.PostID, in.CommentID, in.Reason, board.ReportKindUser)
		if err != nil {
			if strings.HasPrefix(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create report.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		d.notifyBoardManagers(r.Context(), courseCode, boardID, "board_moderation_report",
			"Board content reported",
			"A board post or comment was reported and needs review.")
		telemetry.RecordBusinessEvent("board.moderation.report_created")
		_ = b
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(reportJSON(*created))
	}
}

// handleResolveBoardReport is POST .../reports/{report_id}/resolve
func (d Deps) handleResolveBoardReport() http.HandlerFunc {
	type reqBody struct {
		Action string `json:"action"` // dismiss | hide | remove | resolve
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		reportID := chi.URLParam(r, "report_id")
		_, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to moderate this board.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		action := strings.ToLower(strings.TrimSpace(in.Action))
		status := board.ReportStatusResolved
		if action == "dismiss" {
			status = board.ReportStatusDismissed
		}

		// Load report first so we can act on the target.
		reports, err := board.ListOpenReports(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load report.")
			return
		}
		var target *board.Report
		for i := range reports {
			if reports[i].ID == reportID {
				target = &reports[i]
				break
			}
		}
		if target == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report not found.")
			return
		}

		switch action {
		case "hide":
			if target.PostID != nil {
				_, _ = board.SetPostHidden(r.Context(), d.Pool, courseCode, boardID, *target.PostID, true)
			}
			if target.CommentID != nil && target.PostID != nil {
				_, _ = board.SoftHideComment(r.Context(), d.Pool, courseCode, boardID, *target.PostID, *target.CommentID)
			}
		case "remove":
			if target.PostID != nil && target.CommentID == nil {
				_, _ = board.SoftRemovePost(r.Context(), d.Pool, courseCode, boardID, *target.PostID)
			}
			if target.CommentID != nil && target.PostID != nil {
				_, _ = board.SoftHideComment(r.Context(), d.Pool, courseCode, boardID, *target.PostID, *target.CommentID)
			}
		case "dismiss", "resolve", "":
			// status already set
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "action must be dismiss, hide, remove, or resolve.")
			return
		}

		resolved, err := board.ResolveReport(r.Context(), d.Pool, courseCode, boardID, reportID, viewer, status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not resolve report.")
			return
		}
		if resolved == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report not found.")
			return
		}
		tid, _ := uuid.Parse(reportID)
		_ = board.InsertModerationLog(r.Context(), d.Pool, boardID, &viewer, board.ModActionReportResolve, board.TargetReport, &tid, action+": "+in.Reason)
		telemetry.RecordBusinessEvent("board.moderation.report_resolved")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(reportJSON(*resolved))
	}
}

// handleGetBoardModerationLog is GET .../boards/{board_id}/moderation/log
func (d Deps) handleGetBoardModerationLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		_, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view the moderation log.")
			return
		}
		entries, err := board.ListModerationLog(r.Context(), d.Pool, courseCode, boardID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load moderation log.")
			return
		}
		out := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			out = append(out, moderationLogJSON(e))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}
