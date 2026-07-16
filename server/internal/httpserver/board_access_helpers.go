package httpserver

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func (d Deps) resolveBoardAccess(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	viewer uuid.UUID,
	b *board.Board,
) (board.Capabilities, bool) {
	opts := board.ResolveOpts{
		CourseCode:             courseCode,
		ExternalSharingAllowed: d.effectiveConfig().FFBoardsExternalSharing,
	}
	if d.effectiveConfig().CoppaWorkflowEnabled {
		hasMinors, err := board.CourseHasEnrolledMinors(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to evaluate access policy.")
			return board.Capabilities{}, false
		}
		opts.ForbidExternalForMinors = hasMinors
	}
	caps, err := board.ResolveAccess(r.Context(), d.Pool, b, viewer, opts)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve board access.")
		return board.Capabilities{}, false
	}
	if !caps.CanView {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
		return board.Capabilities{}, false
	}
	return caps, true
}

func (d Deps) loadBoardWithAccess(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	viewer uuid.UUID,
	boardID string,
) (*board.Board, board.Capabilities, bool) {
	b, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load board.")
		return nil, board.Capabilities{}, false
	}
	if b == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
		return nil, board.Capabilities{}, false
	}
	caps, ok := d.resolveBoardAccess(w, r, courseCode, viewer, b)
	if !ok {
		return nil, board.Capabilities{}, false
	}
	return b, caps, true
}

func (d Deps) externalSharingAllowedForCourse(w http.ResponseWriter, r *http.Request, courseCode string) (bool, string, bool) {
	cfg := d.effectiveConfig()
	blocked, reason, err := board.ExternalSharingBlocked(
		r.Context(), d.Pool, courseCode, cfg.FFBoardsExternalSharing, cfg.CoppaWorkflowEnabled,
	)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to evaluate sharing policy.")
		return false, "", false
	}
	if blocked {
		return false, reason, true
	}
	return true, "", true
}

func applyAuthorVisibility(out map[string]any, authorID *string, guestName string, attribution string, caps board.Capabilities) {
	if board.RevealAuthor(attribution, caps) {
		if authorID != nil {
			out["authorId"] = *authorID
		} else {
			out["authorId"] = nil
		}
		if guestName != "" {
			out["guestDisplayName"] = guestName
		}
		return
	}
	out["authorId"] = nil
	if attribution == board.AttributionNamed && guestName != "" {
		out["guestDisplayName"] = guestName
	}
}

func boardJSONWithAccess(b board.Board, caps board.Capabilities) map[string]any {
	out := boardJSON(b)
	out["capabilities"] = map[string]any{
		"canView":     caps.CanView,
		"canPost":     caps.CanPost,
		"canInteract": caps.CanInteract,
		"canArrange":  caps.CanArrange,
		"canManage":   caps.CanManage,
	}
	return out
}

func recordBoardAccessChange(event string) {
	telemetry.RecordBusinessEvent(event)
}
