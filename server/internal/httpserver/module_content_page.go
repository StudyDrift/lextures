package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/relativeschedule"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	ctrepo "github.com/lextures/lextures/server/internal/repos/coursetranslation"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	rlrepo "github.com/lextures/lextures/server/internal/repos/readinglevel"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	lpsvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

type dueAtPatchMode uint8

const (
	dueAtPatchOmit dueAtPatchMode = iota
	dueAtPatchClear
	dueAtPatchSet
)

func parseDueAtJSON(raw json.RawMessage) (dueAtPatchMode, time.Time, error) {
	if len(raw) == 0 {
		return dueAtPatchOmit, time.Time{}, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return dueAtPatchOmit, time.Time{}, err
	}
	if v == nil {
		return dueAtPatchClear, time.Time{}, nil
	}
	s, ok := v.(string)
	if !ok {
		return dueAtPatchOmit, time.Time{}, fmt.Errorf("dueAt must be a string or null")
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return dueAtPatchOmit, time.Time{}, err
	}
	return dueAtPatchSet, t.UTC(), nil
}

func buildModuleContentPageGetResponse(itemID uuid.UUID, row *coursemodulecontent.CourseItemContentRow, shift *relativeschedule.Context) moduleAssignmentGetResponse {
	bFalse := false
	sText := true
	sFile := false
	sURL := false
	lpol := "allow"
	posting := "automatic"
	return moduleAssignmentGetResponse{
		ItemID:                       itemID,
		Title:                        row.Title,
		Markdown:                     row.Markdown,
		DueAt:                        shiftMaybe(shift, row.DueAt),
		UpdatedAt:                    row.UpdatedAt,
		RequiresAssignmentAccessCode: &bFalse,
		SubmissionAllowText:          &sText,
		SubmissionAllowFileUpload:    &sFile,
		SubmissionAllowURL:           &sURL,
		LateSubmissionPolicy:         &lpol,
		BlindGrading:                 false,
		ViewerCanRevealIdentities:    false,
		ModeratedGrading:             false,
		NeverDrop:                    false,
		ReplaceWithFinal:             false,
		PostingPolicy:                &posting,
	}
}

// handleGetModuleContentPage is GET /api/v1/courses/{course_code}/content-pages/{item_id}.
func (d Deps) handleGetModuleContentPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
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
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			visible, err := coursestructure.ContentPageVisibleToStudent(
				r.Context(), d.Pool, *cid, itemID, viewer, time.Now().UTC(),
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check content page access.")
				return
			}
			if !visible {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
				return
			}
			if !d.enforceConditionalRelease(w, r, *cid, viewer, itemID, canEdit) {
				return
			}
			d.recordConditionalReleaseView(r, *cid, viewer, itemID)
		}
		row, err := coursemodulecontent.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load content page.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		var shift *relativeschedule.Context
		if !canEdit {
			shift, err = relativeschedule.LoadForUser(r.Context(), d.Pool, *cid, viewer)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course schedule.")
				return
			}
		}
		out := buildModuleContentPageGetResponse(itemID, row, shift)
		d.enrichModuleItemResponse(r, *cid, itemID, rlrepo.TypeContentPage, viewer, canEdit, &out)
		d.enrichModuleItemWithTranslation(r.Context(), *cid, itemID, ctrepo.TypeContentPage, viewer, canEdit, &out)
		d.enrichIntroCourseContentPage(r, courseCode, *cid, itemID, viewer, canEdit, &out.Title, &out.Markdown)
		if !canEdit && d.profileAdaptEnabled("modality") {
			adaptive, err := d.loadAdaptiveContext(r.Context(), viewer)
			if err == nil {
				pref, err := lpsvc.ResolveModalityPreference(r.Context(), d.Pool, adaptive, *cid, itemID)
				if err == nil {
					out.ProfileRationale = rationaleToJSON(pref.Rationale)
					out.PreferredAlternateItemID = pref.PreferredItemID
					if len(pref.Alternates) > 0 {
						out.ModalityAlternates = make([]modalityAlternateJSON, 0, len(pref.Alternates))
						for _, alt := range pref.Alternates {
							out.ModalityAlternates = append(out.ModalityAlternates, modalityAlternateJSON{
								ItemID: alt.ItemID, Title: alt.Title, Modality: alt.Modality,
							})
						}
					}
				}
			}
		}
		// AC.6 — student adaptive content serving (non-blocking; base on any miss).
		if !canEdit {
			d.applyAdaptiveContentServing(r, *cid, itemID, viewer, &out)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// applyAdaptiveContentServing resolves ACE serving for a student content-page GET (AC.6).
// Failures never block the page: students always receive base content at worst.
func (d Deps) applyAdaptiveContentServing(r *http.Request, courseID, itemID, viewer uuid.UUID, out *moduleAssignmentGetResponse) {
	if d.Pool == nil || out == nil {
		return
	}
	enabled, err := acrepo.AdaptiveContentEnabledForCourse(r.Context(), d.Pool, courseID)
	if err != nil || !acsvc.ActiveForCourse(enabled) {
		return
	}

	// Gateway evaluate for COPPA / opt-out / consent (no model call — FR-7).
	// Deny ⇒ base content with no adapted indicator.
	gatewayAllowed := true
	if _, blocked := d.evaluateAIGatewayBlock(r.Context(), viewer, aigateway.FeatureAdaptiveContent, "", "adaptive-content-serve"); blocked {
		gatewayAllowed = false
	}

	baseMarkdown := out.Markdown
	res := acsvc.ResolveServing(r.Context(), d.Pool, acsvc.ServeRequest{
		CourseID:          courseID,
		BaseContentItemID: itemID,
		UserID:            viewer,
		BaseMarkdown:      baseMarkdown,
		CourseFlag:        enabled,
		GatewayAllowed:    gatewayAllowed,
		EnqueueOnMiss:     true,
	})
	if !res.Applicable {
		return
	}

	// When adapted, swap markdown and ship base as originalMarkdown for "View original".
	if res.IsAdapted {
		out.Markdown = res.Markdown
		if res.OriginalMarkdown != "" {
			om := res.OriginalMarkdown
			out.OriginalMarkdown = &om
		}
	}

	axes := res.AxesApplied
	if axes == nil {
		axes = []string{}
	}
	optoutAllowed := true
	if settings, err := acrepo.GetSettings(r.Context(), d.Pool, courseID); err == nil && settings != nil {
		optoutAllowed = settings.StudentOptoutAllowed
	}

	out.Adaptive = &moduleAdaptiveServingJSON{
		UnitID:                res.UnitID,
		IsAdapted:             res.IsAdapted,
		ServedVariantID:       res.ServedVariantID,
		AxesApplied:           axes,
		CanViewOriginal:       res.CanViewOriginal,
		OptedOut:              res.OptedOut,
		IsHoldout:             res.IsHoldout,
		WasFallback:           res.WasFallback,
		AdaptationReason:      res.AdaptationReason,
		PreAssessmentItemID:   res.PreAssessmentItemID,
		RequiresPreAssessment: res.RequiresPreAssessment,
		OptoutAllowed:         optoutAllowed,
	}
}

// handlePatchModuleContentPage is PATCH /api/v1/courses/{course_code}/content-pages/{item_id}.
func (d Deps) handlePatchModuleContentPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var req struct {
			Markdown string          `json:"markdown"`
			DueAt    json.RawMessage `json:"dueAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		dueMode, dueVal, err := parseDueAtJSON(req.DueAt)
		if err != nil {
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
		locked, found, err := coursestructure.ItemBlueprintLockState(r.Context(), d.Pool, *cid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify structure item.")
			return
		}
		if found && locked {
			cOrg, err := course.CourseOrgID(r.Context(), d.Pool, courseCode)
			if err != nil || cOrg == nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course.")
				return
			}
			if !d.userCanManageBlueprintLocks(r.Context(), viewer, *cOrg) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This item is managed by the district blueprint.")
				return
			}
		}
		var touchDue bool
		var duePtr *time.Time
		switch dueMode {
		case dueAtPatchOmit:
			touchDue = false
		case dueAtPatchClear:
			touchDue = true
			duePtr = nil
		case dueAtPatchSet:
			touchDue = true
			duePtr = &dueVal
		}
		itemIDCopy := itemID
		req.Markdown = d.maybeReconcileContentToolMarkdown(r.Context(), courseCode, *cid, &itemIDCopy, req.Markdown)
		row, err := coursemodulecontent.PatchContentPage(r.Context(), d.Pool, *cid, itemID, req.Markdown, touchDue, duePtr)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save content page.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		// AC.3/AC.4: base content edits bump content_version, supersede variants, re-enqueue signatures.
		if units, _, err := acrepo.BumpUnitsForBaseContentItemWithIDs(r.Context(), d.Pool, *cid, itemID); err == nil {
			for _, u := range units {
				if u.Status != "active" {
					continue
				}
				if _, err := acsvc.EnqueueRegenForUnit(r.Context(), d.Pool, u); err != nil {
					// best-effort
					_ = err
				}
			}
		}
		if d.readingLevelEnabled() {
			_ = rlrepo.ScoreAndPersist(r.Context(), d.Pool, itemID, rlrepo.TypeContentPage, req.Markdown)
			row, _ = coursemodulecontent.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
		}
		out := buildModuleContentPageGetResponse(itemID, row, nil)
		d.enrichModuleItemResponse(r, *cid, itemID, rlrepo.TypeContentPage, viewer, true, &out)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
