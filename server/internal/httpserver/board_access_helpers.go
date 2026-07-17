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
	pol, ok := d.orgBoardPoliciesForCourse(w, r, courseCode)
	if !ok {
		return board.Capabilities{}, false
	}
	opts := board.ResolveOpts{
		CourseCode:             courseCode,
		ExternalSharingAllowed: board.ExternalSharingAllowed(d.effectiveConfig().FFBoardsExternalSharing, pol),
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
	pol, ok := d.orgBoardPoliciesForCourse(w, r, courseCode)
	if !ok {
		return false, "", false
	}
	cfg := d.effectiveConfig()
	allowed := board.ExternalSharingAllowed(cfg.FFBoardsExternalSharing, pol)
	blocked, reason, err := board.ExternalSharingBlocked(
		r.Context(), d.Pool, courseCode, allowed, cfg.CoppaWorkflowEnabled,
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

func (d Deps) orgBoardPoliciesForCourse(w http.ResponseWriter, r *http.Request, courseCode string) (board.OrgPolicies, bool) {
	orgID, err := board.OrgIDForCourse(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
		return board.OrgPolicies{}, false
	}
	pol, err := board.ResolveOrgPolicies(r.Context(), d.Pool, orgID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load board policies.")
		return board.OrgPolicies{}, false
	}
	return pol, true
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

func (d Deps) boardJSONWithAccessAndPolicies(r *http.Request, courseCode string, b board.Board, caps board.Capabilities) map[string]any {
	out := boardJSONWithAccess(b, caps)
	orgID, err := board.OrgIDForCourse(r.Context(), d.Pool, courseCode)
	if err != nil {
		return out
	}
	pol, err := board.ResolveOrgPolicies(r.Context(), d.Pool, orgID)
	if err != nil {
		return out
	}
	out["externalSharingAllowed"] = board.ExternalSharingAllowed(d.effectiveConfig().FFBoardsExternalSharing, pol)
	out["minorModerationFloor"] = d.minorsModerationFloor(r.Context(), courseCode)
	return out
}

func recordBoardAccessChange(event string) {
	telemetry.RecordBusinessEvent(event)
}
