package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// visualBoardsFeatureOff returns true when boards are disabled for the course.
// Access is controlled only by the per-course visualBoardsEnabled flag (no platform master switch).
func (d Deps) visualBoardsFeatureOff(w http.ResponseWriter, r *http.Request, courseCode string) bool {
	crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || crow == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return true
	}
	if !crow.VisualBoardsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Collaboration boards are not enabled for this course.")
		return true
	}
	return false
}

func boardJSON(b board.Board) map[string]any {
	settings := json.RawMessage(`{}`)
	if len(b.Settings) > 0 {
		settings = b.Settings
	}
	mode := b.ReactionMode
	if mode == "" {
		mode = board.ReactionModeNone
	}
	vis := b.Visibility
	if vis == "" {
		vis = board.VisibilityCourse
	}
	attr := b.Attribution
	if attr == "" {
		attr = board.AttributionNamed
	}
	modMode := b.ModerationMode
	if modMode == "" {
		modMode = board.ModerationOpen
	}
	filterAction := b.FilterAction
	if filterAction == "" {
		filterAction = board.FilterFlag
	}
	out := map[string]any{
		"id":             b.ID,
		"courseId":       b.CourseID,
		"title":          b.Title,
		"description":    b.Description,
		"slug":           b.Slug,
		"archived":       b.Archived,
		"layout":         b.Layout,
		"layoutLocked":   b.LayoutLocked,
		"settings":       settings,
		"reactionMode":   mode,
		"visibility":     vis,
		"attribution":    attr,
		"canPost":        b.CanPost,
		"canInteract":    b.CanInteract,
		"canArrange":     b.CanArrange,
		"moderationMode": modMode,
		"filterAction":   filterAction,
		"locked":         b.Locked,
		"createdBy":      b.CreatedBy,
		"createdAt":      b.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":      b.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if b.VisibilityTarget != nil {
		out["visibilityTarget"] = *b.VisibilityTarget
	} else {
		out["visibilityTarget"] = nil
	}
	if b.AssignmentID != nil {
		out["assignmentId"] = *b.AssignmentID
	} else {
		out["assignmentId"] = nil
	}
	if b.FrozenUntil != nil {
		out["frozenUntil"] = b.FrozenUntil.UTC().Format(time.RFC3339)
	} else {
		out["frozenUntil"] = nil
	}
	return out
}

// handleListBoards is GET /api/v1/courses/{course_code}/boards.
func (d Deps) handleListBoards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
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
		includeArchived := strings.EqualFold(r.URL.Query().Get("includeArchived"), "true")
		boards, err := board.List(r.Context(), d.Pool, courseCode, includeArchived)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list boards.")
			return
		}
		pol, okPol := d.orgBoardPoliciesForCourse(w, r, courseCode)
		if !okPol {
			return
		}
		externalOK := board.ExternalSharingAllowed(d.effectiveConfig().FFBoardsExternalSharing, pol)
		out := make([]map[string]any, 0, len(boards))
		for _, b := range boards {
			caps, err := board.ResolveAccess(r.Context(), d.Pool, &b, viewer, board.ResolveOpts{
				CourseCode:             courseCode,
				ExternalSharingAllowed: externalOK,
			})
			if err != nil || !caps.CanView {
				continue
			}
			out = append(out, boardJSONWithAccess(b, caps))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"boards": out})
	}
}

// handleCreateBoard is POST /api/v1/courses/{course_code}/boards.
// Optional ?from=template:{id} or ?from=board:{id}&mode=structure|full (VC.8).
func (d Deps) handleCreateBoard() http.HandlerFunc {
	type reqBody struct {
		Title       string `json:"title"`
		Description string `json:"description"`
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
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		pol, ok := d.orgBoardPoliciesForCourse(w, r, courseCode)
		if !ok {
			return
		}
		exceeded, capErr := board.BoardCapExceeded(r.Context(), d.Pool, courseCode, pol)
		if capErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not verify board limit.")
			return
		}
		if exceeded {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden,
				"This course has reached the organization board limit.")
			telemetry.RecordBusinessEvent("board.create.cap_rejected")
			return
		}
		from := strings.TrimSpace(r.URL.Query().Get("from"))
		if from != "" {
			if d.createBoardFromSelector(w, r, courseCode, viewer, in.Title, in.Description, from) {
				return
			}
		}
		created, err := board.Create(r.Context(), d.Pool, courseCode, viewer, in.Title, in.Description)
		if err != nil {
			if strings.Contains(err.Error(), "title") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create board.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		created = d.applyCreatePolicies(r, courseCode, created, pol)
		telemetry.RecordBusinessEvent("board.created")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardJSON(*created))
	}
}

// applyCreatePolicies sets org default attribution and minors moderation floor on a new board (VC.10).
func (d Deps) applyCreatePolicies(r *http.Request, courseCode string, created *board.Board, pol board.OrgPolicies) *board.Board {
	if created == nil {
		return created
	}
	attr := pol.DefaultAttribution
	if attr == "" {
		attr = board.AttributionNamed
	}
	mode, filter := board.ApplyMinorsModerationFloor(created.ModerationMode, created.FilterAction, d.minorsModerationFloor(r.Context(), courseCode))
	patched, err := board.Patch(r.Context(), d.Pool, courseCode, created.ID, board.PatchBoardInput{
		Attribution:    &attr,
		ModerationMode: &mode,
		FilterAction:   &filter,
	})
	if err != nil || patched == nil {
		return created
	}
	return patched
}

// handleGetBoard is GET /api/v1/courses/{course_code}/boards/{board_id}.
func (d Deps) handleGetBoard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
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
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(d.boardJSONWithAccessAndPolicies(r, courseCode, *b, caps))
	}
}

// handlePatchBoard is PATCH /api/v1/courses/{course_code}/boards/{board_id}.
func (d Deps) handlePatchBoard() http.HandlerFunc {
	type reqBody struct {
		Title            *string         `json:"title"`
		Description      *string         `json:"description"`
		Archived         *bool           `json:"archived"`
		Layout           *string         `json:"layout"`
		LayoutLocked     *bool           `json:"layoutLocked"`
		Settings         json.RawMessage `json:"settings"`
		ReactionMode     *string         `json:"reactionMode"`
		AssignmentID     *string         `json:"assignmentId"`
		Visibility       *string         `json:"visibility"`
		VisibilityTarget *string         `json:"visibilityTarget"`
		Attribution      *string         `json:"attribution"`
		CanPost          *bool           `json:"canPost"`
		CanInteract      *bool           `json:"canInteract"`
		CanArrange       *bool           `json:"canArrange"`
		ModerationMode   *string         `json:"moderationMode"`
		FilterAction     *string         `json:"filterAction"`
		Locked           *bool           `json:"locked"`
		FrozenUntil      *string         `json:"frozenUntil"`
		FreezeMinutes    *int            `json:"freezeMinutes"`
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
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		boardID := chi.URLParam(r, "board_id")
		if in.AssignmentID != nil && strings.TrimSpace(*in.AssignmentID) != "" {
			aid := strings.TrimSpace(*in.AssignmentID)
			var okItem bool
			err := d.Pool.QueryRow(r.Context(), `
				SELECT EXISTS (
					SELECT 1
					FROM course.course_structure_items i
					INNER JOIN course.courses c ON c.id = i.course_id
					WHERE i.id = $1::uuid AND c.course_code = $2
				)
			`, aid, courseCode).Scan(&okItem)
			if err != nil || !okItem {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "assignmentId must be a structure item in this course.")
				return
			}
		}
		if in.Visibility != nil {
			vis, err := board.NormalizeVisibility(*in.Visibility)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			if vis == board.VisibilityLink || vis == board.VisibilityPublic {
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
			}
			if err := board.ValidateVisibilityTarget(r.Context(), d.Pool, courseCode, vis, in.VisibilityTarget); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			// Clear target when switching away from section/group.
			if vis != board.VisibilitySection && vis != board.VisibilityGroup && in.VisibilityTarget == nil {
				empty := ""
				in.VisibilityTarget = &empty
			}
		}

		existing, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}

		modMode := in.ModerationMode
		filterAction := in.FilterAction
		if d.minorsModerationFloor(r.Context(), courseCode) {
			// Org floor: instructors cannot loosen below approval + block.
			forcedMode := board.ModerationApproval
			forcedFilter := board.FilterBlock
			if modMode != nil && *modMode != board.ModerationApproval {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden,
					"Approval mode is required for courses with minors and cannot be disabled.")
				return
			}
			if filterAction != nil && *filterAction != board.FilterBlock {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden,
					"Blocking content filter is required for courses with minors and cannot be relaxed.")
				return
			}
			modMode = &forcedMode
			filterAction = &forcedFilter
		}

		frozenUntil := in.FrozenUntil
		if in.FreezeMinutes != nil {
			until := freezeMinutesToUntil(parseFreezeMinutes(in.FreezeMinutes), time.Now().UTC())
			s := until.Format(time.RFC3339)
			frozenUntil = &s
		}

		updated, err := board.Patch(r.Context(), d.Pool, courseCode, boardID, board.PatchBoardInput{
			Title:            in.Title,
			Description:      in.Description,
			Archived:         in.Archived,
			Layout:           in.Layout,
			LayoutLocked:     in.LayoutLocked,
			Settings:         in.Settings,
			ReactionMode:     in.ReactionMode,
			AssignmentID:     in.AssignmentID,
			Visibility:       in.Visibility,
			VisibilityTarget: in.VisibilityTarget,
			Attribution:      in.Attribution,
			CanPost:          in.CanPost,
			CanInteract:      in.CanInteract,
			CanArrange:       in.CanArrange,
			ModerationMode:   modMode,
			FilterAction:     filterAction,
			Locked:           in.Locked,
			FrozenUntil:      frozenUntil,
		})
		if err != nil {
			if strings.Contains(err.Error(), "title") || strings.Contains(err.Error(), "layout") ||
				strings.Contains(err.Error(), "settings") || strings.Contains(err.Error(), "reaction_mode") ||
				strings.Contains(err.Error(), "assignment_id") || strings.Contains(err.Error(), "visibility") ||
				strings.Contains(err.Error(), "attribution") || strings.Contains(err.Error(), "moderation_mode") ||
				strings.Contains(err.Error(), "filter_action") || strings.Contains(err.Error(), "frozen_until") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update board.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		if in.Archived != nil && *in.Archived {
			telemetry.RecordBusinessEvent("board.archived")
		}
		if in.Layout != nil {
			telemetry.RecordBusinessEvent("board.layout.changed")
		}
		if in.Visibility != nil || in.Attribution != nil || in.CanPost != nil || in.CanInteract != nil || in.CanArrange != nil {
			recordBoardAccessChange("board.access.changed")
		}
		if in.Locked != nil && *in.Locked != existing.Locked {
			action := board.ModActionLock
			if !*in.Locked {
				action = board.ModActionUnlock
			}
			_ = board.InsertModerationLog(r.Context(), d.Pool, boardID, &viewer, action, board.TargetBoard, nil, "")
			telemetry.RecordBusinessEvent("board.moderation." + action)
		}
		if frozenUntil != nil {
			action := board.ModActionFreeze
			if strings.TrimSpace(*frozenUntil) == "" {
				action = board.ModActionUnfreeze
			}
			_ = board.InsertModerationLog(r.Context(), d.Pool, boardID, &viewer, action, board.TargetBoard, nil, "")
			telemetry.RecordBusinessEvent("board.moderation." + action)
		}
		caps := board.Capabilities{CanView: true, CanPost: true, CanInteract: true, CanArrange: true, CanManage: true}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(d.boardJSONWithAccessAndPolicies(r, courseCode, *updated, caps))
	}
}

// handleDeleteBoard is DELETE /api/v1/courses/{course_code}/boards/{board_id}.
// Soft-archives by default; hard-deletes when ?hard=true and caller has enrollments:update (course manage).
func (d Deps) handleDeleteBoard() http.HandlerFunc {
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
		hard := strings.EqualFold(r.URL.Query().Get("hard"), "true")
		if hard {
			canManage, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":enrollments:update")
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
				return
			}
			if !canManage {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Hard delete requires course manage permission.")
				return
			}
			okDel, err := board.HardDelete(r.Context(), d.Pool, courseCode, boardID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete board.")
				return
			}
			if !okDel {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
				return
			}
			telemetry.RecordBusinessEvent("board.deleted")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		updated, err := board.SoftDelete(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not archive board.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.archived")
		w.WriteHeader(http.StatusNoContent)
	}
}
