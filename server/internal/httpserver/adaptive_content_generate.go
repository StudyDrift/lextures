package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/concepts"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
	"github.com/lextures/lextures/server/internal/service/contentpagegeneration"
)

// registerAdaptiveContentGenerateRoutes is called from registerAdaptiveContentRoutes (AC.3).
func (d Deps) registerAdaptiveContentGenerateRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/variants/preview", d.handleAdaptiveContentVariantPreview())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/variants", d.handleAdaptiveContentVariantsList())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/key-terms", d.handleAdaptiveContentKeyTermsList())
	r.Put("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/key-terms", d.handleAdaptiveContentKeyTermsPut())
}

// handleAdaptiveContentVariantPreview is POST .../units/{id}/variants/preview (instructor).
func (d Deps) handleAdaptiveContentVariantPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
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
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load unit.")
			return
		}
		if unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}

		var body acmodel.PreviewVariantRequest
		if r.Body != nil {
			dec := json.NewDecoder(r.Body)
			_ = dec.Decode(&body) // empty body is ok
		}

		base, err := coursemodulecontent.GetForCourseItem(r.Context(), d.Pool, *cid, unit.BaseContentItemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load base content.")
			return
		}
		if base == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Base content page not found for this unit.")
			return
		}

		profile, err := d.resolvePreviewProfile(r.Context(), *unit, body)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		// Cache hit for non-neutral signatures (FR-7).
		persist := true
		if body.Persist != nil {
			persist = *body.Persist
		}
		if !profile.IsNeutral && profile.ProfileSignature != acsvc.NeutralSignature {
			cached, err := acrepo.GetVariantBySignature(r.Context(), d.Pool, unit.ID, profile.ProfileSignature, unit.ContentVersion)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load variant cache.")
				return
			}
			if cached != nil && cached.Status != "rejected" && cached.Status != "superseded" {
				acsvc.IncCacheHit()
				out := variantRowToAPI(*cached)
				out.CacheHit = true
				// Rebuild sections from markdown for preview UI.
				out.Sections = []acmodel.DraftSection{{Heading: "", Markdown: cached.VariantMarkdown}}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(acmodel.PreviewVariantResponse{
					Variant:          out,
					FidelityScore:    ptrFloat(cached.FidelityScore),
					A11yFlags:        acrepo.ParseFlagsJSON(cached.A11yFlags),
					SafetyFlags:      acrepo.ParseFlagsJSON(cached.SafetyFlags),
					PromptTokens:     int(cached.PromptTokens),
					CompletionTokens: int(cached.CompletionTokens),
					BaseMarkdown:     base.Markdown,
				})
				return
			}
		}

		keyTerms, err := acrepo.ListKeyTerms(r.Context(), d.Pool, unit.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load key terms.")
			return
		}
		termStrs := make([]string, 0, len(keyTerms))
		for _, kt := range keyTerms {
			if kt.MustAppear {
				termStrs = append(termStrs, kt.Term)
			}
		}

		conceptLabels, misLabels := d.loadACELabels(r.Context(), *cid, profile)

		settings, _ := acrepo.GetSettings(r.Context(), d.Pool, *cid)
		requireApproval := false
		if settings != nil {
			requireApproval = settings.RequireInstructorApproval
		}

		// Effective axes: unit override or course defaults.
		axes := unit.AllowedAxes
		if len(axes) == 0 && settings != nil {
			axes = settings.AllowedAxes
		}

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) && !profile.IsNeutral && profile.ProfileSignature != acsvc.NeutralSignature {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}

		sysPrompt := acsvc.DefaultSystemPrompt
		if s, err := systemprompts.GetByKey(r.Context(), d.Pool, acsvc.PromptKey); err == nil && strings.TrimSpace(s) != "" {
			sysPrompt = s
		}

		// Gateway evaluation (FR-6). Neutral short-circuit skips gateway/model.
		gatewayAllowed := true
		if !profile.IsNeutral && profile.ProfileSignature != acsvc.NeutralSignature {
			promptMaterial := base.Markdown + profile.ProfileSignature + profile.EmphasisMode
			if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureAdaptiveContent, model, promptMaterial) {
				// enforceAIGateway already wrote the response.
				// Also log a blocked usage path via existing helper; return without generating.
				return
			}
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		genIn := acsvc.GenerateInput{
			BaseMarkdown:              base.Markdown,
			BaseTitle:                 base.Title,
			Profile:                   profile,
			AllowedAxes:               axes,
			KeyTerms:                  termStrs,
			ConceptLabels:             conceptLabels,
			MisconceptionLabels:       misLabels,
			Model:                     model,
			SystemPrompt:              sysPrompt,
			PromptVersion:             acsvc.PromptVersionCurrent,
			MinFidelity:               unit.MinFidelity,
			ContentVersion:            unit.ContentVersion,
			RequireInstructorApproval: requireApproval,
			GatewayAllowed:            gatewayAllowed,
		}

		variant, callMeta, genErr := acsvc.GenerateVariant(r.Context(), bound, genIn)

		// Always log usage when a model was involved (success or structured reject with tokens).
		if callMeta.Usage.HasData() || (genErr == nil && !variant.Fallback) || (variant.PromptTokens+variant.CompletionTokens > 0) {
			d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureAdaptiveContent, model, string(callMeta.Provider), base.Markdown+profile.ProfileSignature, aigateway.Decision{OptInConfirmed: true})
			d.recordAIProviderUsage(r.Context(), AIUsageMeta{
				UserID: viewer, CourseID: cid, CourseCode: courseCode, Feature: aigateway.FeatureAdaptiveContent, Model: model,
			}, callMeta, genErr == nil || errors.Is(genErr, acsvc.ErrRejectedFidelity) || errors.Is(genErr, acsvc.ErrRejectedSafety))
		}

		// Hard failures (gateway already handled; model/config).
		if genErr != nil && !errors.Is(genErr, acsvc.ErrRejectedFidelity) && !errors.Is(genErr, acsvc.ErrRejectedSafety) &&
			!errors.Is(genErr, acsvc.ErrGatewayDenied) && !errors.Is(genErr, acsvc.ErrBudgetExhausted) {
			if errors.Is(genErr, acsvc.ErrGenerationFailed) {
				// Fallback to base in the response body (preview still useful).
			} else {
				writeAIGenerationFailed(w, r, "Adaptive content generation failed: "+genErr.Error(), genErr)
				return
			}
		}

		// Persist for cache / audit when requested and we have a signature (FR-4 stores rejected too).
		var storedID uuid.UUID
		if persist && variant.ProfileSignature != "" && !variant.IsNeutralLike() {
			fid := variant.FidelityScore
			row := acrepo.VariantRow{
				UnitID:           unit.ID,
				ProfileSignature: variant.ProfileSignature,
				AxesApplied:      variant.AxesApplied,
				VariantMarkdown:  variant.Markdown,
				Model:            variant.Model,
				FidelityScore:    &fid,
				SafetyFlags:      acrepo.FlagsJSON(variant.SafetyFlags),
				Status:           variant.Status,
				PromptVersion:    variant.PromptVersion,
				ContentVersion:   variant.ContentVersion,
				PromptTokens:     int32(variant.PromptTokens),
				CompletionTokens: int32(variant.CompletionTokens),
				A11yFlags:        acrepo.FlagsJSON(variant.A11yFlags),
			}
			// Don't overwrite auto_served/approved with rejected on a pure preview of a bad gen? Plan says store rejected for audit.
			saved, err := acrepo.UpsertVariant(r.Context(), d.Pool, row)
			if err == nil && saved != nil {
				storedID = saved.ID
			}
			eventType := acsvc.EventVariantGenerated
			if variant.Status == "rejected" {
				eventType = acsvc.EventVariantRejected
			}
			actor := viewer
			_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &unit.ID, &actor, nil, eventType, map[string]any{
				"profileSignature": variant.ProfileSignature,
				"status":           variant.Status,
				"fidelityScore":    variant.FidelityScore,
				"fallbackReason":   variant.FallbackReason,
				"model":            variant.Model,
				"promptVersion":    variant.PromptVersion,
			})
		}

		apiVar := acmodel.ContentVariant{
			ID:               storedID,
			UnitID:           unit.ID,
			ProfileSignature: variant.ProfileSignature,
			AxesApplied:      variant.AxesApplied,
			Sections:         draftSectionsToAPI(variant.Sections),
			VariantMarkdown:  variant.Markdown,
			Model:            variant.Model,
			FidelityScore:    &variant.FidelityScore,
			SafetyFlags:      nonNilStrings(variant.SafetyFlags),
			A11yFlags:        nonNilStrings(variant.A11yFlags),
			Status:           variant.Status,
			PromptVersion:    variant.PromptVersion,
			ContentVersion:   variant.ContentVersion,
			PromptTokens:     int32(variant.PromptTokens),
			CompletionTokens: int32(variant.CompletionTokens),
			Fallback:         variant.Fallback,
			FallbackReason:   variant.FallbackReason,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.PreviewVariantResponse{
			Variant:          apiVar,
			FidelityScore:    variant.FidelityScore,
			A11yFlags:        nonNilStrings(variant.A11yFlags),
			SafetyFlags:      nonNilStrings(variant.SafetyFlags),
			PromptTokens:     variant.PromptTokens,
			CompletionTokens: variant.CompletionTokens,
			BaseMarkdown:     base.Markdown,
		})
	}
}

// handleAdaptiveContentVariantsList is GET .../units/{id}/variants (instructor|reviewer).
func (d Deps) handleAdaptiveContentVariantsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, _, ok := d.requireAdaptiveContentReview(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
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
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load unit.")
			return
		}
		if unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
		var rows []acrepo.VariantRow
		var listErr error
		if statusFilter != "" {
			rows, listErr = acrepo.ListVariantsByStatus(r.Context(), d.Pool, unitID, statusFilter)
		} else {
			rows, listErr = acrepo.ListVariants(r.Context(), d.Pool, unitID)
		}
		if listErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list variants.")
			return
		}
		out := make([]acmodel.ContentVariant, 0, len(rows))
		for _, row := range rows {
			out = append(out, variantRowToAPI(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.VariantsListResponse{Variants: out})
	}
}

// handleAdaptiveContentKeyTermsList is GET .../units/{id}/key-terms (instructor).
func (d Deps) handleAdaptiveContentKeyTermsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
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
		terms, err := acrepo.ListKeyTerms(r.Context(), d.Pool, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list key terms.")
			return
		}
		out := make([]acmodel.KeyTerm, 0, len(terms))
		for _, t := range terms {
			out = append(out, acmodel.KeyTerm{
				ID: t.ID, UnitID: t.UnitID, Term: t.Term, MustAppear: t.MustAppear, CreatedAt: t.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.KeyTermsListResponse{KeyTerms: out})
	}
}

// handleAdaptiveContentKeyTermsPut is PUT .../units/{id}/key-terms (instructor).
func (d Deps) handleAdaptiveContentKeyTermsPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
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
		var body acmodel.PutKeyTermsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		rows := make([]acrepo.KeyTermRow, 0, len(body.Terms))
		for _, t := range body.Terms {
			term := strings.TrimSpace(t.Term)
			if term == "" {
				continue
			}
			must := true
			if t.MustAppear != nil {
				must = *t.MustAppear
			}
			rows = append(rows, acrepo.KeyTermRow{UnitID: unitID, Term: term, MustAppear: must})
		}
		if err := acrepo.ReplaceKeyTerms(r.Context(), d.Pool, unitID, rows); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save key terms.")
			return
		}
		actor := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &unitID, &actor, nil, "key_terms_updated", map[string]any{
			"count": len(rows),
		})
		terms, err := acrepo.ListKeyTerms(r.Context(), d.Pool, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list key terms.")
			return
		}
		out := make([]acmodel.KeyTerm, 0, len(terms))
		for _, t := range terms {
			out = append(out, acmodel.KeyTerm{
				ID: t.ID, UnitID: t.UnitID, Term: t.Term, MustAppear: t.MustAppear, CreatedAt: t.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.KeyTermsListResponse{KeyTerms: out})
	}
}

func (d Deps) resolvePreviewProfile(ctx context.Context, unit acrepo.UnitRow, body acmodel.PreviewVariantRequest) (acsvc.ProfileResult, error) {
	sig := strings.TrimSpace(body.ProfileSignature)
	if sig == acsvc.NeutralSignature || strings.EqualFold(sig, "neutral") {
		return acsvc.NeutralProfile(unit.ID, unit.AllowedAxes, "default", "default"), nil
	}
	if body.SyntheticProfile != nil {
		sp := body.SyntheticProfile
		gaps := make([]acsvc.ConceptGap, 0, len(sp.ConceptGaps))
		for _, g := range sp.ConceptGaps {
			gaps = append(gaps, acsvc.ConceptGap{ConceptID: g.ConceptID, Gap: g.Gap})
		}
		return acsvc.SyntheticProfileFromRequest(
			sp.EmphasisMode, sp.TargetBloom, sp.ReadingLevelPref, sp.ModalityPref,
			gaps, sp.Misconceptions, sp.AxisSet,
		), nil
	}
	if sig != "" {
		// Load any stored profile with this signature for payload reconstruction.
		// Cohort may have many enrollments; pick the first matching payload.
		// Fall back to a minimal profile with that signature.
		return acsvc.ProfileResult{
			EmphasisMode:     acsvc.EmphasisReinforce,
			TargetBloom:      "understand",
			ProfileSignature: sig,
			IsNeutral:        false,
			ReadingLevelPref: "default",
			ModalityPref:     "default",
			AxisSet:          unit.AllowedAxes,
			Payload:          acsvc.ProfilePayload{},
		}, nil
	}
	// Default synthetic: reinforce.
	return acsvc.SyntheticProfileFromRequest(
		acsvc.EmphasisReinforce, "understand", "default", "default",
		nil, nil, unit.AllowedAxes,
	), nil
}

func (d Deps) loadACELabels(ctx context.Context, courseID uuid.UUID, profile acsvc.ProfileResult) (map[uuid.UUID]string, map[string]string) {
	conceptLabels := map[uuid.UUID]string{}
	ids := make([]uuid.UUID, 0, len(profile.Payload.ConceptGaps))
	for _, g := range profile.Payload.ConceptGaps {
		ids = append(ids, g.ConceptID)
	}
	if len(ids) > 0 {
		if rows, err := concepts.LoadConceptsByIDs(ctx, d.Pool, ids); err == nil {
			for _, c := range rows {
				conceptLabels[c.ID] = c.Name
			}
		}
	}
	misLabels := map[string]string{}
	if len(profile.Payload.Misconceptions) > 0 {
		// Best-effort name lookup via course misconceptions table.
		for _, mid := range profile.Payload.Misconceptions {
			id, err := uuid.Parse(mid)
			if err != nil {
				misLabels[mid] = mid
				continue
			}
			var name string
			err = d.Pool.QueryRow(ctx, `
SELECT name FROM course.misconceptions WHERE id = $1 AND course_id = $2
`, id, courseID).Scan(&name)
			if err != nil || name == "" {
				misLabels[mid] = mid
			} else {
				misLabels[mid] = name
			}
		}
	}
	return conceptLabels, misLabels
}

func variantRowToAPI(row acrepo.VariantRow) acmodel.ContentVariant {
	return acmodel.ContentVariant{
		ID:               row.ID,
		UnitID:           row.UnitID,
		ProfileSignature: row.ProfileSignature,
		AxesApplied:      row.AxesApplied,
		VariantMarkdown:  row.VariantMarkdown,
		Model:            row.Model,
		FidelityScore:    row.FidelityScore,
		SafetyFlags:      acrepo.ParseFlagsJSON(row.SafetyFlags),
		A11yFlags:        acrepo.ParseFlagsJSON(row.A11yFlags),
		Status:           row.Status,
		PromptVersion:    row.PromptVersion,
		ContentVersion:   row.ContentVersion,
		PromptTokens:     row.PromptTokens,
		CompletionTokens: row.CompletionTokens,
		CreatedAt:        row.CreatedAt,
		HumanEdited:      row.HumanEdited,
		ReviewedBy:       row.ReviewedBy,
		ReviewedAt:       row.ReviewedAt,
		ReviewNote:       row.ReviewNote,
		VariantVersion:   row.VariantVersion,
		ApprovedBy:       row.ApprovedBy,
	}
}

func ptrFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func draftSectionsToAPI(in []contentpagegeneration.DraftSection) []acmodel.DraftSection {
	out := make([]acmodel.DraftSection, 0, len(in))
	for _, s := range in {
		out = append(out, acmodel.DraftSection{Heading: s.Heading, Markdown: s.Markdown})
	}
	return out
}
