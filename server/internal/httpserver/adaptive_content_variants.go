package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

// registerAdaptiveContentAuthoringRoutes is AC.5 review/approval surface.
func (d Deps) registerAdaptiveContentAuthoringRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/adaptive-content/review-queue", d.handleAdaptiveContentReviewQueue())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/variants/{variant_id}/approve", d.handleAdaptiveContentVariantApprove())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/variants/{variant_id}/reject", d.handleAdaptiveContentVariantReject())
	r.Put("/api/v1/courses/{course_code}/adaptive-content/variants/{variant_id}", d.handleAdaptiveContentVariantEditApprove())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/variants/{variant_id}/revoke", d.handleAdaptiveContentVariantRevoke())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/variants/bulk", d.handleAdaptiveContentVariantsBulk())
}

// requireAdaptiveContentReview allows instructors (item:create) or TA review-only capability.
func (d Deps) requireAdaptiveContentReview(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, canConfigure bool, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, false, false
	}
	canConfigure, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, false, false
	}
	if canConfigure {
		return courseCode, viewer, true, true
	}
	canReview, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":adaptive_content:review")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, false, false
	}
	if !canReview {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor or adaptive content review permission required.")
		return "", uuid.UUID{}, false, false
	}
	return courseCode, viewer, false, true
}

// handleAdaptiveContentReviewQueue is GET .../adaptive-content/review-queue (instructor|reviewer).
func (d Deps) handleAdaptiveContentReviewQueue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, _, ok := d.requireAdaptiveContentReview(w, r)
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
		limit := 50
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				offset = n
			}
		}
		rows, total, err := acrepo.ListPendingReviewForCourse(r.Context(), d.Pool, *cid, limit, offset)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list review queue.")
			return
		}
		out := make([]acmodel.ContentVariant, 0, len(rows))
		for _, row := range rows {
			out = append(out, variantRowToAPI(row))
		}
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.ReviewQueueResponse{
			Variants: out,
			Total:    total,
			Limit:    limit,
			Offset:   offset,
		})
	}
}

// handleAdaptiveContentVariantApprove is POST .../variants/{vid}/approve.
func (d Deps) handleAdaptiveContentVariantApprove() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, _, ok := d.requireAdaptiveContentReview(w, r)
		if !ok {
			return
		}
		variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid variant id.")
			return
		}
		var body acmodel.ReviewVariantRequest
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		out, err := d.applyVariantReview(r, courseCode, viewer, variantID, "approve", body, nil)
		if err != nil {
			writeVariantReviewError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdaptiveContentVariantReject is POST .../variants/{vid}/reject.
func (d Deps) handleAdaptiveContentVariantReject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, _, ok := d.requireAdaptiveContentReview(w, r)
		if !ok {
			return
		}
		variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid variant id.")
			return
		}
		var body acmodel.ReviewVariantRequest
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		out, err := d.applyVariantReview(r, courseCode, viewer, variantID, "reject", body, nil)
		if err != nil {
			writeVariantReviewError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdaptiveContentVariantEditApprove is PUT .../variants/{vid} (edit-and-approve).
func (d Deps) handleAdaptiveContentVariantEditApprove() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, _, ok := d.requireAdaptiveContentReview(w, r)
		if !ok {
			return
		}
		variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid variant id.")
			return
		}
		var body acmodel.EditAndApproveVariantRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		md := strings.TrimSpace(body.VariantMarkdown)
		if md == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrEmptyVariantMarkdown.Error())
			return
		}
		rev := acmodel.ReviewVariantRequest{
			ExpectedVariantVersion: body.ExpectedVariantVersion,
			Note:                   body.Note,
			OverrideGate:           body.OverrideGate,
		}
		out, err := d.applyVariantReview(r, courseCode, viewer, variantID, "approve", rev, &md)
		if err != nil {
			writeVariantReviewError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdaptiveContentVariantRevoke is POST .../variants/{vid}/revoke (instructor only — not review-only).
func (d Deps) handleAdaptiveContentVariantRevoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		// Revoke requires full instructor (item:create), not TA review-only.
		courseCode, viewer, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid variant id.")
			return
		}
		var body acmodel.ReviewVariantRequest
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
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
		existing, err := acrepo.GetVariantByID(r.Context(), d.Pool, *cid, variantID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load variant.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Variant not found.")
			return
		}
		if !acsvc.CanRevokeStatus(existing.Status) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrVariantNotRevocable.Error())
			return
		}
		note := acsvc.ReviewNotePtr(body.Note)
		out, err := acrepo.RevokeVariant(r.Context(), d.Pool, variantID, viewer, body.ExpectedVariantVersion, note)
		if err != nil {
			if errors.Is(err, acrepo.ErrVariantVersionConflict) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Variant was modified by another reviewer; reload and try again.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke variant.")
			return
		}
		if out == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Variant not found.")
			return
		}
		actor := viewer
		unitID := out.UnitID
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &unitID, &actor, nil, acsvc.EventVariantRevoked, map[string]any{
			"variantId":        out.ID,
			"profileSignature": out.ProfileSignature,
			"previousStatus":   existing.Status,
			"status":           out.Status,
			"note":             note,
		})
		acsvc.IncRevoked()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(variantRowToAPI(*out))
	}
}

// handleAdaptiveContentVariantsBulk is POST .../units/{id}/variants/bulk.
func (d Deps) handleAdaptiveContentVariantsBulk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, _, ok := d.requireAdaptiveContentReview(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
			return
		}
		var body acmodel.BulkVariantsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := acsvc.ValidateBulkAction(body.Action, len(body.VariantIDs)); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
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
		if err != nil || unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		action := strings.ToLower(strings.TrimSpace(body.Action))
		results := make([]acmodel.BulkVariantRow, 0, len(body.VariantIDs))
		succeeded := 0
		failed := 0
		for _, vid := range body.VariantIDs {
			existing, err := acrepo.GetVariantByID(r.Context(), d.Pool, *cid, vid)
			if err != nil {
				failed++
				results = append(results, acmodel.BulkVariantRow{VariantID: vid, OK: false, Error: "failed to load variant"})
				continue
			}
			if existing == nil {
				failed++
				results = append(results, acmodel.BulkVariantRow{VariantID: vid, OK: false, Error: "variant not found"})
				continue
			}
			if existing.UnitID != unitID {
				failed++
				results = append(results, acmodel.BulkVariantRow{VariantID: vid, OK: false, Error: "variant does not belong to this unit"})
				continue
			}
			rev := acmodel.ReviewVariantRequest{
				ExpectedVariantVersion: body.ExpectedVariantVersion,
				Note:                   body.Note,
				OverrideGate:           body.OverrideGate,
			}
			out, err := d.applyVariantReview(r, courseCode, viewer, vid, action, rev, nil)
			if err != nil {
				failed++
				results = append(results, acmodel.BulkVariantRow{
					VariantID: vid,
					OK:        false,
					Error:     err.Error(),
				})
				continue
			}
			succeeded++
			results = append(results, acmodel.BulkVariantRow{
				VariantID: vid,
				OK:        true,
				Status:    out.Status,
			})
		}
		actor := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &unitID, &actor, nil, acsvc.EventVariantsBulk, map[string]any{
			"action":    action,
			"succeeded": succeeded,
			"failed":    failed,
			"count":     len(body.VariantIDs),
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.BulkVariantsResponse{
			Succeeded: succeeded,
			Failed:    failed,
			Results:   results,
		})
	}
}

func (d Deps) applyVariantReview(
	r *http.Request,
	courseCode string,
	viewer uuid.UUID,
	variantID uuid.UUID,
	action string,
	body acmodel.ReviewVariantRequest,
	editedMarkdown *string,
) (acmodel.ContentVariant, error) {
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		return acmodel.ContentVariant{}, err
	}
	if cid == nil {
		return acmodel.ContentVariant{}, errNotFound("Course not found.")
	}
	existing, err := acrepo.GetVariantByID(r.Context(), d.Pool, *cid, variantID)
	if err != nil {
		return acmodel.ContentVariant{}, err
	}
	if existing == nil {
		return acmodel.ContentVariant{}, errNotFound("Variant not found.")
	}
	unit, err := acrepo.GetUnit(r.Context(), d.Pool, *cid, existing.UnitID)
	if err != nil {
		return acmodel.ContentVariant{}, err
	}
	if unit == nil {
		return acmodel.ContentVariant{}, errNotFound("Unit not found.")
	}

	action = strings.ToLower(strings.TrimSpace(action))
	note := acsvc.ReviewNotePtr(body.Note)
	md := existing.VariantMarkdown
	humanEdited := false
	if editedMarkdown != nil {
		md = *editedMarkdown
		humanEdited = true
	}

	// Re-run hard key-term check on the body that will be stored.
	mustTerms, err := acrepo.MustAppearKeyTerms(r.Context(), d.Pool, existing.UnitID)
	if err != nil {
		return acmodel.ContentVariant{}, err
	}
	safetyFlags := acrepo.ParseFlagsJSON(existing.SafetyFlags)
	a11yFlags := acrepo.ParseFlagsJSON(existing.A11yFlags)

	switch action {
	case "approve":
		if !acsvc.CanApproveStatus(existing.Status) {
			return acmodel.ContentVariant{}, acsvc.ErrVariantNotPending
		}
		// Hard key-term failures cannot be overridden (AC.5 AC-6).
		if okTerms, missing := acsvc.CheckKeyTerms(md, mustTerms); !okTerms {
			return acmodel.ContentVariant{}, &reviewBlockedError{
				msg: acsvc.ErrHardKeyTermFailure.Error() + ": " + strings.Join(missing, ", "),
			}
		}
		// Soft gate: fidelity/safety may be overridden with audit.
		if acsvc.SoftGateFailed(existing.FidelityScore, unit.MinFidelity, safetyFlags, a11yFlags) && !body.OverrideGate {
			return acmodel.ContentVariant{}, acsvc.ErrGateFailedNoOverride
		}
		// If human-edited, re-check soft fidelity score floor only when score is known and no override.
		// (We do not re-run the LLM judge on edit.)
	case "reject":
		if !acsvc.CanRejectStatus(existing.Status) {
			return acmodel.ContentVariant{}, acsvc.ErrVariantNotPending
		}
	default:
		return acmodel.ContentVariant{}, acsvc.ErrInvalidReviewAction
	}

	newStatus := "approved"
	if action == "reject" {
		newStatus = "rejected"
	}

	dec := acrepo.ReviewDecision{
		NewStatus:       newStatus,
		ExpectedVersion: body.ExpectedVariantVersion,
		Markdown:        editedMarkdown,
		ReviewNote:      note,
		Actor:           viewer,
		HumanEdited:     humanEdited,
		OverrideGate:    body.OverrideGate,
	}
	out, err := acrepo.ApplyReviewDecision(r.Context(), d.Pool, variantID, dec)
	if err != nil {
		if errors.Is(err, acrepo.ErrVariantVersionConflict) {
			return acmodel.ContentVariant{}, errVersionConflict
		}
		return acmodel.ContentVariant{}, err
	}
	if out == nil {
		return acmodel.ContentVariant{}, errNotFound("Variant not found.")
	}

	actor := viewer
	unitID := out.UnitID
	eventType := acsvc.EventVariantApproved
	if action == "reject" {
		eventType = acsvc.EventVariantRejectedByReview
	} else if humanEdited {
		eventType = acsvc.EventVariantEdited
	}
	_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &unitID, &actor, nil, eventType, map[string]any{
		"variantId":        out.ID,
		"profileSignature": out.ProfileSignature,
		"previousStatus":   existing.Status,
		"status":           out.Status,
		"humanEdited":      out.HumanEdited,
		"overrideGate":     body.OverrideGate && action == "approve",
		"note":             note,
		"beforeVersion":    existing.VariantVersion,
		"afterVersion":     out.VariantVersion,
	})

	if action == "approve" {
		if humanEdited {
			acsvc.IncEdited()
		}
		acsvc.IncApproved()
	} else {
		acsvc.IncRejected()
	}
	acsvc.ObserveTimeInQueue(acsvc.TimeInQueueMs(existing.CreatedAt))

	return variantRowToAPI(*out), nil
}

type reviewBlockedError struct{ msg string }

func (e *reviewBlockedError) Error() string { return e.msg }

type simpleMsgError struct {
	msg  string
	kind string // not_found | conflict
}

func (e *simpleMsgError) Error() string { return e.msg }

func errNotFound(msg string) error {
	return &simpleMsgError{msg: msg, kind: "not_found"}
}

var errVersionConflict = &simpleMsgError{msg: "Variant was modified by another reviewer; reload and try again.", kind: "conflict"}

func writeVariantReviewError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var blocked *reviewBlockedError
	if errors.As(err, &blocked) {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, blocked.msg)
		return
	}
	var simple *simpleMsgError
	if errors.As(err, &simple) {
		if simple.kind == "not_found" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, simple.msg)
			return
		}
		if simple.kind == "conflict" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, simple.msg)
			return
		}
	}
	if errors.Is(err, acrepo.ErrVariantVersionConflict) {
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Variant was modified by another reviewer; reload and try again.")
		return
	}
	if errors.Is(err, acsvc.ErrHardKeyTermFailure) ||
		errors.Is(err, acsvc.ErrGateFailedNoOverride) ||
		errors.Is(err, acsvc.ErrVariantNotPending) ||
		errors.Is(err, acsvc.ErrVariantNotRevocable) ||
		errors.Is(err, acsvc.ErrEmptyVariantMarkdown) ||
		errors.Is(err, acsvc.ErrInvalidReviewAction) ||
		errors.Is(err, acsvc.ErrBulkEmpty) ||
		errors.Is(err, acsvc.ErrBulkTooLarge) {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
		return
	}
	apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to review variant.")
}
