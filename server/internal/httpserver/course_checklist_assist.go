package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	userai "github.com/lextures/lextures/server/internal/repos/user"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/assessmentoutcomesmapping"
	"github.com/lextures/lextures/server/internal/service/welcomedraft"
)

// handleSuggestCourseOutcomeLinks is POST /api/v1/courses/{course_code}/outcomes/suggest-links.
// Returns proposals only — never writes course_outcome_links (CC.10 FR-6 / AC-3).
func (d Deps) handleSuggestCourseOutcomeLinks() http.HandlerFunc {
	type resp struct {
		Proposals []assessmentoutcomesmapping.Proposal `json:"proposals"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			// Staff fallback for designers/teachers with staff enrollment but nonstandard perms.
			isStaff, staffErr := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
			if staffErr != nil || !isStaff {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to suggest outcome mappings.")
				return
			}
		}

		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		outcomes, err := courseoutcomes.ListOutcomes(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load outcomes.")
			return
		}
		if len(outcomes) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Add learning outcomes before suggesting mappings.")
			return
		}

		links, err := courseoutcomes.ListLinksForCourse(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load outcome links.")
			return
		}
		mapped := make(map[uuid.UUID]struct{}, len(links))
		for _, l := range links {
			mapped[l.StructureItemID] = struct{}{}
		}

		rows, err := coursestructure.ListForCourse(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course structure.")
			return
		}
		titleByID := make(map[uuid.UUID]string, len(rows))
		for _, row := range rows {
			titleByID[row.ID] = row.Title
		}

		var unmapped []assessmentoutcomesmapping.AssessmentInput
		for _, row := range rows {
			if row.Archived {
				continue
			}
			if row.Kind != "assignment" && row.Kind != "quiz" {
				continue
			}
			if _, ok := mapped[row.ID]; ok {
				continue
			}
			modTitle := ""
			if row.ParentID != nil {
				modTitle = titleByID[*row.ParentID]
			}
			unmapped = append(unmapped, assessmentoutcomesmapping.AssessmentInput{
				ID:     row.ID.String(),
				Title:  row.Title,
				Kind:   row.Kind,
				Module: modTitle,
			})
			if len(unmapped) >= assessmentoutcomesmapping.MaxItems {
				break
			}
		}
		if len(unmapped) == 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(resp{Proposals: []assessmentoutcomesmapping.Proposal{}})
			return
		}

		// Enrich points when available (no learner data).
		var assignIDs []uuid.UUID
		for _, a := range unmapped {
			if a.Kind == "assignment" {
				if id, err := uuid.Parse(a.ID); err == nil {
					assignIDs = append(assignIDs, id)
				}
			}
		}
		if pts, err := coursestructure.LoadAssignmentPointsWorth(ctx, d.Pool, *cid, assignIDs); err == nil {
			for i := range unmapped {
				if id, err := uuid.Parse(unmapped[i].ID); err == nil {
					if p, ok := pts[id]; ok {
						unmapped[i].Points = p
					}
				}
			}
		}

		outcomeInputs := make([]assessmentoutcomesmapping.OutcomeInput, 0, len(outcomes))
		for _, o := range outcomes {
			outcomeInputs = append(outcomeInputs, assessmentoutcomesmapping.OutcomeInput{
				ID:          o.ID.String(),
				Title:       o.Title,
				Description: o.Description,
			})
		}

		courseTitle, courseLang := courseTitleAndLanguage(ctx, d, *cid)

		orgID := d.orgIDPtrForUser(ctx, viewer)
		if !d.aiConfigured(ctx, orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		model, err := userai.GetCourseSetupModelID(ctx, d.Pool, viewer)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}

		meterMaterial := courseTitle
		for _, o := range outcomeInputs {
			meterMaterial += "\n" + o.Title
		}
		for _, a := range unmapped {
			meterMaterial += "\n" + a.Title
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureAssessmentOutcomeMapping, model, meterMaterial) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		proposals, callMeta, err := assessmentoutcomesmapping.Suggest(
			ctx, bound, model, assessmentoutcomesmapping.DefaultSystemPrompt,
			assessmentoutcomesmapping.SuggestInput{
				CourseTitle: courseTitle,
				CourseLang:  courseLang,
				Outcomes:    outcomeInputs,
				Assessments: unmapped,
			},
		)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "parse assessment outcome mapping JSON") {
				writeAIGenerationFailed(w, r, "AI did not return valid outcome mapping JSON: "+msg, err)
				return
			}
			if strings.Contains(msg, "too long") || strings.Contains(msg, "learner data") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureAssessmentOutcomeMapping, model, string(callMeta.Provider), meterMaterial, gwDec)
		d.recordAIProviderUsage(ctx, AIUsageMeta{
			UserID: viewer, CourseID: cid, CourseCode: courseCode, Feature: aigateway.FeatureAssessmentOutcomeMapping, Model: model,
		}, callMeta, true)

		if proposals == nil {
			proposals = []assessmentoutcomesmapping.Proposal{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Proposals: proposals})
	}
}

// handleDraftWelcomeAnnouncement is POST /api/v1/courses/{course_code}/feed/draft-welcome.
// Returns a draft subject+body only — never posts (CC.10 FR-8).
func (d Deps) handleDraftWelcomeAnnouncement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}

		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		title, desc, start, end, lang := courseWelcomeContext(ctx, d, *cid)
		if strings.TrimSpace(title) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Course title is required before drafting a welcome.")
			return
		}

		orgID := d.orgIDPtrForUser(ctx, viewer)
		if !d.aiConfigured(ctx, orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		model, err := userai.GetCourseSetupModelID(ctx, d.Pool, viewer)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}
		meterMaterial := title + "\n" + desc
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureWelcomeDraft, model, meterMaterial) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		draft, callMeta, err := welcomedraft.Generate(ctx, bound, model, welcomedraft.DefaultSystemPrompt, welcomedraft.Input{
			CourseTitle:       title,
			CourseDescription: desc,
			StartDate:         start,
			EndDate:           end,
			Language:          lang,
		})
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "parse welcome draft JSON") {
				writeAIGenerationFailed(w, r, "AI did not return a valid welcome draft: "+msg, err)
				return
			}
			if strings.Contains(msg, "learner data") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureWelcomeDraft, model, string(callMeta.Provider), meterMaterial, gwDec)
		d.recordAIProviderUsage(ctx, AIUsageMeta{
			UserID: viewer, CourseID: cid, CourseCode: courseCode, Feature: aigateway.FeatureWelcomeDraft, Model: model,
		}, callMeta, true)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(draft)
	}
}

func courseTitleAndLanguage(ctx context.Context, d Deps, courseID uuid.UUID) (title, lang string) {
	var t string
	var l *string
	err := d.Pool.QueryRow(ctx, `
SELECT COALESCE(title, course_code), NULLIF(TRIM(catalog_language), '')
FROM course.courses WHERE id = $1
`, courseID).Scan(&t, &l)
	if err != nil {
		return "", ""
	}
	title = t
	if l != nil {
		lang = *l
	}
	return title, lang
}

func courseWelcomeContext(ctx context.Context, d Deps, courseID uuid.UUID) (title, desc, start, end, lang string) {
	var t string
	var description *string
	var language *string
	var startAt, endAt *time.Time
	err := d.Pool.QueryRow(ctx, `
SELECT COALESCE(title, course_code), description, starts_at, ends_at, NULLIF(TRIM(catalog_language), '')
FROM course.courses WHERE id = $1
`, courseID).Scan(&t, &description, &startAt, &endAt, &language)
	if err != nil {
		return "", "", "", "", ""
	}
	title = t
	if description != nil {
		desc = *description
	}
	if language != nil {
		lang = *language
	}
	if startAt != nil {
		start = startAt.UTC().Format("2006-01-02")
	}
	if endAt != nil {
		end = endAt.UTC().Format("2006-01-02")
	}
	return title, desc, start, end, lang
}
