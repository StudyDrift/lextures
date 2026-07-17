package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/service/boardfilter"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const boardReportRateLimit = 10
const boardReportRateWindow = 10 * time.Minute

func (d Deps) writeGateReject(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, board.ErrBoardLocked) {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This board is locked and cannot be edited.")
		return true
	}
	if errors.Is(err, board.ErrBoardFrozen) {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Posting is temporarily frozen on this board.")
		return true
	}
	return false
}

func (d Deps) minorsModerationFloor(ctx context.Context, courseCode string) bool {
	orgID, err := board.OrgIDForCourse(ctx, d.Pool, courseCode)
	if err != nil {
		return false
	}
	pol, err := board.ResolveOrgPolicies(ctx, d.Pool, orgID)
	if err != nil || !pol.MinorModerationFloor {
		return false
	}
	if !d.effectiveConfig().CoppaWorkflowEnabled {
		return false
	}
	hasMinors, err := board.CourseHasEnrolledMinors(ctx, d.Pool, courseCode)
	return err == nil && hasMinors
}

// screenBoardText runs the content filter. On block, writes 400 and returns false.
// On flag, returns matched=true so the caller can create a filter report.
func (d Deps) screenBoardText(
	w http.ResponseWriter,
	r *http.Request,
	b *board.Board,
	actorID *uuid.UUID,
	title string,
	body []byte,
) (matched bool, term string, ok bool) {
	plain := boardfilter.ExtractPlainText(title, body)
	res := boardfilter.Match(plain, nil)
	if !res.Matched {
		return false, "", true
	}
	action := b.FilterAction
	if action == "" {
		action = board.FilterFlag
	}
	_ = board.InsertModerationLog(r.Context(), d.Pool, b.ID, actorID, board.ModActionFilterHit, board.TargetBoard, nil, res.Term)
	telemetry.RecordBusinessEvent("board.moderation.filter_hit")
	if action == board.FilterBlock {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This content could not be posted. Please revise and try again.")
		return true, res.Term, false
	}
	return true, res.Term, true
}

func (d Deps) notifyBoardManagers(ctx context.Context, courseCode, boardID, eventType, title, body string) {
	actionURL := "/courses/" + courseCode + "/boards/" + boardID + "?moderation=1"
	rows, err := d.Pool.Query(ctx, `
		SELECT ce.user_id
		FROM course.course_enrollments ce
		INNER JOIN course.courses c ON c.id = ce.course_id
		INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_staff = true
		WHERE c.course_code = $1 AND ce.status = 'active'
	`, courseCode)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var staffID uuid.UUID
		if scanErr := rows.Scan(&staffID); scanErr != nil {
			continue
		}
		_, _ = notificationsinbox.Insert(ctx, d.Pool, staffID, eventType, title, body, actionURL)
	}
}

func (d Deps) flagFilterHit(
	ctx context.Context,
	courseCode string,
	b *board.Board,
	postID string,
	term string,
) {
	pid := postID
	reason := "Content filter match"
	if term != "" {
		reason = "Content filter match: " + term
	}
	_, _ = board.CreateReport(ctx, d.Pool, courseCode, b.ID, nil, &pid, nil, reason, board.ReportKindFilter)
	d.notifyBoardManagers(ctx, courseCode, b.ID, "board_moderation_filter",
		"Board content flagged",
		"A post was flagged by the content filter and needs review.")
}

func (d Deps) flagAVBlockedAttachment(ctx context.Context, courseCode string, b *board.Board, postID string) {
	pid := postID
	_, _ = board.CreateReport(ctx, d.Pool, courseCode, b.ID, nil, &pid, nil, "Attachment blocked by antivirus scan", board.ReportKindAVBlocked)
	actorNil := (*uuid.UUID)(nil)
	tid, _ := uuid.Parse(postID)
	_ = board.InsertModerationLog(ctx, d.Pool, b.ID, actorNil, board.ModActionAVBlocked, board.TargetPost, &tid, "scan_status=blocked")
	d.notifyBoardManagers(ctx, courseCode, b.ID, "board_moderation_av",
		"Board attachment blocked",
		"An attachment was blocked by scanning and needs review.")
	telemetry.RecordBusinessEvent("board.moderation.av_blocked")
}

func resolveInitialPostStatus(b *board.Board, isManager bool) string {
	if isManager {
		return board.PostStatusApproved
	}
	if b.ModerationMode == board.ModerationApproval {
		return board.PostStatusPending
	}
	return board.PostStatusApproved
}

func reportJSON(r board.Report) map[string]any {
	out := map[string]any{
		"id":        r.ID,
		"boardId":   r.BoardID,
		"reason":    r.Reason,
		"kind":      r.Kind,
		"status":    r.Status,
		"createdAt": r.CreatedAt.UTC().Format(time.RFC3339),
	}
	if r.PostID != nil {
		out["postId"] = *r.PostID
	}
	if r.CommentID != nil {
		out["commentId"] = *r.CommentID
	}
	if r.ReporterID != nil {
		out["reporterId"] = *r.ReporterID
	}
	if r.ResolvedAt != nil {
		out["resolvedAt"] = r.ResolvedAt.UTC().Format(time.RFC3339)
	}
	if r.ResolvedBy != nil {
		out["resolvedBy"] = *r.ResolvedBy
	}
	return out
}

func moderationLogJSON(e board.ModerationLogEntry) map[string]any {
	out := map[string]any{
		"id":         e.ID,
		"boardId":    e.BoardID,
		"action":     e.Action,
		"targetType": e.TargetType,
		"reason":     e.Reason,
		"createdAt":  e.CreatedAt.UTC().Format(time.RFC3339),
	}
	if e.ActorID != nil {
		out["actorId"] = *e.ActorID
	}
	if e.TargetID != nil {
		out["targetId"] = *e.TargetID
	}
	return out
}

func freezeMinutesToUntil(minutes int, now time.Time) time.Time {
	if minutes <= 0 {
		minutes = 5
	}
	if minutes > 60 {
		minutes = 60
	}
	return now.Add(time.Duration(minutes) * time.Minute)
}

func parseFreezeMinutes(raw *int) int {
	if raw == nil {
		return 5
	}
	return *raw
}
