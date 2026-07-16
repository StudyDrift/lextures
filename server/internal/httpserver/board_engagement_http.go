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
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const boardCommentRateLimit = 30
const boardCommentRateWindow = 10 * time.Minute

func boardCommentJSON(c board.Comment) map[string]any {
	return boardCommentJSONWithAttribution(c, board.AttributionNamed, board.Capabilities{CanManage: true})
}

func boardCommentJSONWithAttribution(c board.Comment, attribution string, caps board.Capabilities) map[string]any {
	out := map[string]any{
		"id":        c.ID,
		"postId":    c.PostID,
		"body":      json.RawMessage(c.Body),
		"hidden":    c.Hidden,
		"createdAt": c.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt": c.UpdatedAt.UTC().Format(time.RFC3339),
	}
	applyAuthorVisibility(out, c.AuthorID, "", attribution, caps)
	if c.ParentID != nil {
		out["parentId"] = *c.ParentID
	}
	return out
}

func mergePostEngagement(out map[string]any, e board.PostEngagement, mode string) {
	out["commentCount"] = e.CommentCount
	switch mode {
	case board.ReactionModeNone:
		return
	case board.ReactionModeStar:
		out["reactionCount"] = e.ReactionCount
		if e.AvgStars != nil {
			out["avgStars"] = *e.AvgStars
		}
		if e.MyReaction != nil {
			out["myReaction"] = e.MyReaction
		}
	case board.ReactionModeGrade:
		out["reactionCount"] = e.ReactionCount
		if e.Grade != nil {
			out["grade"] = *e.Grade
		}
		if e.MyReaction != nil {
			out["myReaction"] = e.MyReaction
		}
	default: // like, vote
		out["reactionCount"] = e.ReactionCount
		if e.MyReaction != nil {
			out["myReaction"] = e.MyReaction
		}
	}
}

func (d Deps) userCanGradeBoard(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID) (bool, bool) {
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false, false
	}
	return hasPerm, true
}

func (d Deps) loadBoardForEngagement(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID, boardID string) (*board.Board, board.Capabilities, bool) {
	b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
	if !ok {
		return nil, board.Capabilities{}, false
	}
	return b, caps, true
}

// handlePutBoardPostReaction is PUT .../posts/{post_id}/reaction
func (d Deps) handlePutBoardPostReaction() http.HandlerFunc {
	type reqBody struct {
		Kind  string   `json:"kind"`
		Value *float64 `json:"value"`
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
		b, caps, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanInteract {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to interact on this board.")
			return
		}
		if d.writeGateReject(w, board.CheckWriteAllowed(b, caps.CanManage, board.WriteReact, time.Now().UTC())) {
			return
		}
		if b.ReactionMode == board.ReactionModeNone {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Reactions are disabled on this board.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		kind := strings.TrimSpace(strings.ToLower(in.Kind))
		expected := board.ModeToKind(b.ReactionMode)
		if kind == "" {
			kind = expected
		}
		if kind != expected {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Reaction kind does not match board reaction mode.")
			return
		}
		if kind == board.ReactionKindGrade {
			canGrade, ok := d.userCanGradeBoard(w, r, courseCode, viewer)
			if !ok {
				return
			}
			if !canGrade {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to grade cards.")
				return
			}
		}
		p, err := board.GetPost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load post.")
			return
		}
		if p == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		result, err := board.SetReaction(r.Context(), d.Pool, courseCode, boardID, postID, viewer, kind, in.Value)
		if err != nil {
			if strings.Contains(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not set reaction.")
			return
		}
		if result == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.reaction.set")
		canGrade, _ := d.userCanGradeBoard(w, r, courseCode, viewer)
		eng, err := board.LoadPostEngagements(r.Context(), d.Pool, courseCode, boardID, viewer, b.ReactionMode, canGrade)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load aggregates.")
			return
		}
		out := map[string]any{
			"active":  result.Active,
			"removed": result.Removed,
		}
		if e, ok := eng[postID]; ok {
			mergePostEngagement(out, e, b.ReactionMode)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleDeleteBoardPostReaction is DELETE .../posts/{post_id}/reaction
func (d Deps) handleDeleteBoardPostReaction() http.HandlerFunc {
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
		b, caps, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanInteract {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to interact on this board.")
			return
		}
		if d.writeGateReject(w, board.CheckWriteAllowed(b, caps.CanManage, board.WriteReact, time.Now().UTC())) {
			return
		}
		kind := board.ModeToKind(b.ReactionMode)
		if kind == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Reactions are disabled on this board.")
			return
		}
		if kind == board.ReactionKindGrade {
			canGrade, ok := d.userCanGradeBoard(w, r, courseCode, viewer)
			if !ok {
				return
			}
			if !canGrade {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to grade cards.")
				return
			}
		}
		if err := board.DeleteReaction(r.Context(), d.Pool, courseCode, boardID, postID, viewer, kind); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not remove reaction.")
			return
		}
		telemetry.RecordBusinessEvent("board.reaction.cleared")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListBoardPostComments is GET .../posts/{post_id}/comments
func (d Deps) handleListBoardPostComments() http.HandlerFunc {
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
		b, caps, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		comments, err := board.ListComments(r.Context(), d.Pool, courseCode, boardID, postID, caps.CanManage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list comments.")
			return
		}
		out := make([]map[string]any, 0, len(comments))
		for _, c := range comments {
			out = append(out, boardCommentJSONWithAttribution(c, b.Attribution, caps))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"comments": out})
	}
}

// handleCreateBoardPostComment is POST .../posts/{post_id}/comments
func (d Deps) handleCreateBoardPostComment() http.HandlerFunc {
	type reqBody struct {
		Body     json.RawMessage `json:"body"`
		ParentID *string         `json:"parentId"`
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
		b, caps, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanInteract {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to interact on this board.")
			return
		}
		if d.writeGateReject(w, board.CheckWriteAllowed(b, caps.CanManage, board.WriteComment, time.Now().UTC())) {
			return
		}
		since := time.Now().UTC().Add(-boardCommentRateWindow)
		n, err := board.CountRecentCommentsByUser(r.Context(), d.Pool, courseCode, boardID, viewer, since)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not check rate limit.")
			return
		}
		if n >= boardCommentRateLimit {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Comment rate limit exceeded. Please try again later.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		matched, term, okFilter := d.screenBoardText(w, r, b, &viewer, "", in.Body)
		if !okFilter {
			return
		}
		created, err := board.CreateComment(r.Context(), d.Pool, courseCode, boardID, postID, viewer, in.Body, in.ParentID)
		if err != nil {
			if strings.Contains(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create comment.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		if matched {
			pid, cid := postID, created.ID
			reason := "Content filter match"
			if term != "" {
				reason = "Content filter match: " + term
			}
			_, _ = board.CreateReport(r.Context(), d.Pool, courseCode, boardID, nil, &pid, &cid, reason, board.ReportKindFilter)
			d.notifyBoardManagers(r.Context(), courseCode, boardID, "board_moderation_filter",
				"Board content flagged",
				"A comment was flagged by the content filter and needs review.")
		}
		telemetry.RecordBusinessEvent("board.comment.created")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardCommentJSONWithAttribution(*created, b.Attribution, caps))
	}
}

// handlePatchBoardPostComment is PATCH .../comments/{comment_id}
func (d Deps) handlePatchBoardPostComment() http.HandlerFunc {
	type reqBody struct {
		Body   json.RawMessage `json:"body"`
		Hidden *bool           `json:"hidden"`
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
		_, caps, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanInteract {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to interact on this board.")
			return
		}
		existing, err := board.GetComment(r.Context(), d.Pool, courseCode, boardID, postID, commentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load comment.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Comment not found.")
			return
		}
		canManage, ok := d.userCanGradeBoard(w, r, courseCode, viewer)
		if !ok {
			return
		}
		isAuthor := existing.AuthorID != nil && *existing.AuthorID == viewer.String()
		if !isAuthor && !canManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if in.Hidden != nil && !canManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only managers can hide comments.")
			return
		}
		updated, err := board.PatchComment(r.Context(), d.Pool, courseCode, boardID, postID, commentID, board.PatchCommentInput{
			Body:   in.Body,
			Hidden: in.Hidden,
		})
		if err != nil {
			if strings.Contains(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update comment.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Comment not found.")
			return
		}
		if in.Hidden != nil && *in.Hidden {
			telemetry.RecordBusinessEvent("board.comment.hidden")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(boardCommentJSON(*updated))
	}
}

// handleDeleteBoardPostComment is DELETE .../comments/{comment_id} (soft-hide).
func (d Deps) handleDeleteBoardPostComment() http.HandlerFunc {
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
		_, caps, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanInteract {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to interact on this board.")
			return
		}
		existing, err := board.GetComment(r.Context(), d.Pool, courseCode, boardID, postID, commentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load comment.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Comment not found.")
			return
		}
		canManage, ok := d.userCanGradeBoard(w, r, courseCode, viewer)
		if !ok {
			return
		}
		isAuthor := existing.AuthorID != nil && *existing.AuthorID == viewer.String()
		if !isAuthor && !canManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		updated, err := board.SoftHideComment(r.Context(), d.Pool, courseCode, boardID, postID, commentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete comment.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Comment not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.comment.hidden")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleBoardPostGradeSync is POST .../posts/{post_id}/grade-sync
func (d Deps) handleBoardPostGradeSync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		canGrade, ok := d.userCanGradeBoard(w, r, courseCode, viewer)
		if !ok {
			return
		}
		if !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to sync grades.")
			return
		}
		boardID := chi.URLParam(r, "board_id")
		postID := chi.URLParam(r, "post_id")
		b, _, ok := d.loadBoardForEngagement(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		// Grade sync is manager-only (checked above); view access still required.
		if b.ReactionMode != board.ReactionModeGrade {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Board is not in grade mode.")
			return
		}
		if b.AssignmentID == nil || *b.AssignmentID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Board is not linked to an assignment.")
			return
		}
		p, err := board.GetPost(r.Context(), d.Pool, courseCode, boardID, postID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load post.")
			return
		}
		if p == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Post not found.")
			return
		}
		if p.AuthorID == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Post has no author to grade.")
			return
		}
		// Prefer the caller's grade reaction; fall back to latest grade on the post.
		react, err := board.GetReaction(r.Context(), d.Pool, courseCode, boardID, postID, viewer, board.ReactionKindGrade)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load grade.")
			return
		}
		var points float64
		if react != nil && react.Value != nil {
			points = *react.Value
		} else {
			eng, err := board.LoadPostEngagements(r.Context(), d.Pool, courseCode, boardID, viewer, b.ReactionMode, true)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load grade.")
				return
			}
			e := eng[postID]
			if e.Grade == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No grade set on this card.")
				return
			}
			points = *e.Grade
		}

		courseID, err := uuid.Parse(b.CourseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}
		studentID, err := uuid.Parse(*p.AuthorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid post author.")
			return
		}
		itemID, err := uuid.Parse(*b.AssignmentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		// Ensure assignment belongs to this course.
		var okItem bool
		err = d.Pool.QueryRow(r.Context(), `
			SELECT EXISTS (
				SELECT 1 FROM course.course_structure_items i
				WHERE i.id = $1 AND i.course_id = $2
			)
		`, itemID, courseID).Scan(&okItem)
		if err != nil || !okItem {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Linked assignment not found in this course.")
			return
		}
		var postingPolicy string
		_ = d.Pool.QueryRow(r.Context(), `
			SELECT COALESCE(NULLIF(TRIM(ma.posting_policy), ''), 'automatic')
			FROM course.module_assignments ma
			WHERE ma.structure_item_id = $1
		`, itemID).Scan(&postingPolicy)
		if postingPolicy == "" {
			postingPolicy = "automatic"
		}
		if err := coursegrades.UpsertCell(r.Context(), d.Pool, courseID, studentID, itemID, points, nil, nil, postingPolicy); err != nil {
			telemetry.RecordBusinessEvent("board.grade_sync.failed")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not write grade to gradebook.")
			return
		}
		telemetry.RecordBusinessEvent("board.grade_sync.ok")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"synced":        true,
			"pointsEarned":  points,
			"studentId":     studentID.String(),
			"assignmentId":  itemID.String(),
			"postingPolicy": postingPolicy,
		})
	}
}
